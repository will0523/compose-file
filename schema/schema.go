package schema

//go:generate go-bindata -pkg schema data

import (
	"fmt"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

type portsFormatChecker struct{}

func (checker portsFormatChecker) IsFormat(input interface{}) bool {
	// TODO: implement this
	return true
}

type durationFormatChecker struct{}

func (checker durationFormatChecker) IsFormat(input interface{}) bool {
	str, ok := input.(string)
	if !ok {
		return false
	}
	_, err := time.ParseDuration(str)
	return err == nil
}

func init() {
	gojsonschema.FormatCheckers.Add("expose", portsFormatChecker{})
	gojsonschema.FormatCheckers.Add("ports", portsFormatChecker{})
	gojsonschema.FormatCheckers.Add("duration", durationFormatChecker{})
}

// Validate uses the jsonschema to validate the configuration
func Validate(config map[string]interface{}) error {
	schemaData, err := Asset("data/config_schema_v3.0.json")
	if err != nil {
		return err
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaData))
	dataLoader := gojsonschema.NewGoLoader(config)

	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		return toError(result)
	}

	return nil
}

func toError(result *gojsonschema.Result) error {
	err := getMostSpecificError(result.Errors())
	description := getDescription(err)
	return fmt.Errorf("%s %s", err.Field(), description)
}

func getDescription(err gojsonschema.ResultError) string {
	if err.Type() == "invalid_type" {
		if expectedType, ok := err.Details()["expected"].(string); ok {
			return fmt.Sprintf("must be a %s", humanReadableType(expectedType))
		}
	}

	return err.Description()
}

func humanReadableType(definition string) string {
	if definition[0:1] == "[" {
		allTypes := strings.Split(definition[1:len(definition)-1], ",")
		for i, t := range allTypes {
			allTypes[i] = humanReadableType(t)
		}
		return fmt.Sprintf(
			"%s or %s",
			strings.Join(allTypes[0:len(allTypes)-1], ", "),
			allTypes[len(allTypes)-1],
		)
	}
	if definition == "object" {
		return "mapping"
	}
	if definition == "array" {
		return "list"
	}
	return definition
}

func getMostSpecificError(errors []gojsonschema.ResultError) gojsonschema.ResultError {
	var mostSpecificError gojsonschema.ResultError

	for _, err := range errors {
		if mostSpecificError == nil {
			mostSpecificError = err
		} else if specificity(err) > specificity(mostSpecificError) {
			mostSpecificError = err
		} else if specificity(err) == specificity(mostSpecificError) {
			// Invalid type errors win in a tie-breaker for most specific field name
			if err.Type() == "invalid_type" && mostSpecificError.Type() != "invalid_type" {
				mostSpecificError = err
			}
		}
	}

	return mostSpecificError
}

func specificity(err gojsonschema.ResultError) int {
	return len(strings.Split(err.Field(), "."))
}
