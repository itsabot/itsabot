package main

import (
	"path"

	"github.com/jbrukh/bayesian"
)

func train(c *bayesian.Classifier, s string) error {
	if err := trainClassifier(c, s); err != nil {
		return err
	}
	if err := c.WriteClassesToFile(path.Join("data")); err != nil {
		return err
	}
	return nil
}
