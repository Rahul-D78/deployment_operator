package controllers

import (
	automation "demo/api/v1alpha1"
)

func labels(v *automation.Automation, driver string) map[string]string {
	return map[string]string{
		"app":     "automation",
		"auto_cr": v.Name,
		"driver":  driver,
	}
}
