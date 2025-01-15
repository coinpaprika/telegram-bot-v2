package database

import (
	"database/sql"
	"fmt"
	"log"
)

func SaveMetric(metricName, labelKey, labelValue string, value float64) error {
	query := `
	INSERT OR REPLACE INTO metrics (metric_name, label_key, label_value, metric_value)
	VALUES (?, ?, ?, ?);`
	_, err := DB.Exec(query, metricName, labelKey, labelValue, value)
	if err != nil {
		return fmt.Errorf("failed to save metric: %w", err)
	}
	log.Printf("Metric saved: %s[%s=%s] = %f", metricName, labelKey, labelValue, value)
	return nil
}

func GetMetric(metricName string) (float64, error) {
	var value float64
	query := `
	SELECT metric_value
	FROM metrics
	WHERE metric_name = ? AND label_key IS NULL AND label_value IS NULL;`
	err := DB.QueryRow(query, metricName).Scan(&value)
	if err == sql.ErrNoRows {
		log.Printf("Metric %s not found in the database, defaulting to 0", metricName)
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to get metric %s: %w", metricName, err)
	}
	log.Printf("Metric loaded: %s = %f", metricName, value)
	return value, nil
}

func SaveMetricWithLabels(metricName, labelKey, labelValue string, value float64) error {
	query := `
	INSERT OR REPLACE INTO metrics (metric_name, label_key, label_value, metric_value)
	VALUES (?, ?, ?, ?);`
	_, err := DB.Exec(query, metricName, labelKey, labelValue, value)
	if err != nil {
		return fmt.Errorf("failed to save metric with labels: %w", err)
	}
	log.Printf("Metric with labels saved: %s[%s=%s] = %f", metricName, labelKey, labelValue, value)
	return nil
}

// GetMetricsWithLabels fetches all metrics with labels for a given metric name
func GetMetricsWithLabels(metricName string) (map[string]map[string]float64, error) {
	query := `
	SELECT label_key, label_value, metric_value
	FROM metrics
	WHERE metric_name = ? AND label_key IS NOT NULL AND label_value IS NOT NULL;`

	rows, err := DB.Query(query, metricName)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics with labels: %w", err)
	}
	defer rows.Close()

	metrics := make(map[string]map[string]float64)
	for rows.Next() {
		var labelKey, labelValue string
		var value float64
		if err := rows.Scan(&labelKey, &labelValue, &value); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if _, exists := metrics[labelKey]; !exists {
			metrics[labelKey] = make(map[string]float64)
		}
		metrics[labelKey][labelValue] = value
	}
	return metrics, nil
}
