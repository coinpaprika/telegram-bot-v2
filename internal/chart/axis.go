// MIT License

// Copyright (c) 2022 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package chart

import (
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
	"strings"

	"github.com/golang/freetype/truetype"
)

type axisPainter struct {
	p   *Painter
	opt *AxisOption
}

func NewAxisPainter(p *Painter, opt AxisOption) *axisPainter {
	return &axisPainter{
		p:   p,
		opt: &opt,
	}
}

type AxisOption struct {
	// The theme of chart
	Theme ColorPalette
	// Formatter for y axis text value
	Formatter string
	// The label of axis
	Data []string
	// The boundary gap on both sides of a coordinate axis.
	// Nil or *true means the center part of two axis ticks
	BoundaryGap *bool
	// The flag for show axis, set this to *false will hide axis
	Show *bool
	// The position of axis, it can be 'left', 'top', 'right' or 'bottom'
	Position string
	// Number of segments that the axis is split into. Note that this number serves only as a recommendation.
	SplitNumber int
	// The line color of axis
	StrokeColor Color
	// The line width
	StrokeWidth float64
	// The length of the axis tick
	TickLength int
	// The first axis
	FirstAxis int
	// The margin value of label
	LabelMargin int
	// The font size of label
	FontSize float64
	// The font of label
	Font *truetype.Font
	// The color of label
	FontColor Color
	// The flag for show axis split line, set this to true will show axis split line
	SplitLineShow bool
	// The color of split line
	SplitLineColor Color
	// The text rotation of label
	TextRotation float64
	// The offset of label
	LabelOffset Box
	Unit        int
}

func (a *axisPainter) Render() (Box, error) {
	opt := a.opt
	top := a.p
	theme := opt.Theme
	if theme == nil {
		theme = top.theme
	}
	if isFalse(opt.Show) {
		return BoxZero, nil
	}

	// Setup font and color configurations
	strokeWidth := opt.StrokeWidth
	if strokeWidth == 0 {
		strokeWidth = 1
	}
	font := opt.Font
	if font == nil {
		font = a.p.font
	}
	if font == nil {
		font = theme.GetFont()
	}
	fontColor := opt.FontColor
	if fontColor.IsZero() {
		fontColor = theme.GetTextColor()
	}
	fontSize := opt.FontSize
	if fontSize == 0 {
		fontSize = theme.GetFontSize()
	}
	strokeColor := opt.StrokeColor
	if strokeColor.IsZero() {
		strokeColor = theme.GetAxisStrokeColor()
	}

	// Process the label data and formatting
	data := opt.Data
	formatter := opt.Formatter
	if len(formatter) != 0 {
		for index, text := range data {
			data[index] = strings.ReplaceAll(formatter, "{value}", text)
		}
	}
	boundaryGap := true
	if isFalse(opt.BoundaryGap) {
		boundaryGap = false
	}
	isVertical := opt.Position == PositionLeft || opt.Position == PositionRight
	labelPosition := ""
	if !boundaryGap {
		labelPosition = PositionLeft
	}
	if isVertical && boundaryGap {
		labelPosition = PositionCenter
	}

	// Configure padding and alignment
	labelPaddingLeft, labelPaddingTop, labelPaddingRight := 0, 0, 0
	textAlign := AlignCenter
	orient := OrientHorizontal
	switch opt.Position {
	case PositionTop:
		labelPaddingTop = 0
		orient = OrientHorizontal
		textAlign = AlignCenter
	case PositionLeft:
		orient = OrientVertical
		textAlign = AlignRight
		labelPaddingRight = 5
	case PositionRight:
		orient = OrientVertical
		textAlign = AlignLeft
		labelPaddingLeft = 5
	case PositionBottom:
		labelPaddingTop = 30 // Adjust padding to move labels down
		labelPaddingLeft = 20
		labelPaddingRight = 20
	}

	tickLength := getDefaultInt(opt.TickLength, 5)

	// Set drawing styles and prepare for rendering
	style := Style{
		StrokeColor: strokeColor,
		StrokeWidth: strokeWidth,
		Font:        font,
		FontColor:   fontColor,
		FontSize:    fontSize,
	}
	top.SetDrawingStyle(style).OverrideTextStyle(style)

	isTextRotation := opt.TextRotation != 0
	if isTextRotation {
		top.SetTextRotation(opt.TextRotation)
	}
	textMaxWidth, textMaxHeight := top.MeasureTextMaxWidthHeight(data)
	if isTextRotation {
		top.ClearTextRotation()
	}

	// Calculate spacing and fit count
	textFillWidth := float64(textMaxWidth + 20)
	fitTextCount := ceilFloatToInt(float64(top.Width()) / textFillWidth)
	unit := opt.Unit
	if unit <= 0 {
		unit = ceilFloatToInt(float64(len(data)) / float64(fitTextCount))
		unit = chart.MaxInt(unit, opt.SplitNumber)
		if unit%2 == 0 && len(data)%(unit+1) == 0 {
			unit++
		}
	}

	// Set dimensions based on orientation
	width := 0
	height := 0
	if isVertical {
		width = textMaxWidth + tickLength<<1
		height = top.Height()
	} else {
		width = top.Width()
		height = tickLength<<1 + textMaxHeight
	}

	// Adjust padding and box configuration
	padding := Box{}
	switch opt.Position {
	case PositionTop:
		padding.Top = top.Height() - height
	case PositionLeft:
		padding.Right = top.Width() - width
	case PositionRight:
		padding.Left = top.Width() - width
	default:
		padding.Top = top.Height() - defaultXAxisHeight
	}

	p := top.Child(PainterPaddingOption(padding))

	// Filter out unwanted labels ("-")
	filteredData := []string{}
	for _, label := range data {
		if label != "-" {
			filteredData = append(filteredData, label)
		}
	}

	// Set `tickCount` to match `filteredData` length
	tickCount := len(filteredData)

	// Draw axis labels with appropriate offsets
	p.Child(PainterPaddingOption(Box{
		Left:  labelPaddingLeft,
		Top:   labelPaddingTop, // Adjust based on alignment needs
		Right: labelPaddingRight,
	})).MultiText(MultiTextOption{
		First:        opt.FirstAxis,
		Align:        textAlign,
		TextList:     filteredData,
		Orient:       orient,
		Unit:         1,
		Position:     labelPosition,
		TextRotation: opt.TextRotation,
		Offset:       Box{Top: -10}, // Reduce Top offset to align better
	})

	// Render vertical grid lines if necessary
	if !isVertical && opt.Position == PositionBottom {
		gridLineStyle := Style{
			StrokeColor: drawing.Color{R: 200, G: 200, B: 200, A: 255},
			StrokeWidth: 1.0,
		}
		p.OverrideDrawingStyle(gridLineStyle)
		step := width / (tickCount - 1)
		for i := 0; i < tickCount; i++ {
			x := step * i
			p.LineStroke([]Point{
				{X: x, Y: height - labelPaddingTop},
				{X: x, Y: 0},
			})
		}
	}

	return Box{
		Bottom: height,
		Right:  width,
	}, nil
}
