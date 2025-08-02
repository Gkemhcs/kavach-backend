package org

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// TestCase represents a single test case from JSON
type TestCase struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Expected    map[string]interface{} `json:"expected"`
	Setup       *TestSetup             `json:"setup,omitempty"`
}

// TestSetup represents setup data for a test case
type TestSetup struct {
	ExistingOrganizations []MockOrganizationSetup `json:"existing_organizations,omitempty"`
}

// MockOrganizationSetup represents organization setup data
type MockOrganizationSetup struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     string `json:"owner_id"`
}

// TestData represents the structure of test data files
type TestData struct {
	TestCases []TestCase `json:"test_cases"`
}

// TestLoader loads test data from JSON files
type TestLoader struct {
	testDataPath string
}

// NewTestLoader creates a new test loader
func NewTestLoader(testDataPath string) *TestLoader {
	return &TestLoader{
		testDataPath: testDataPath,
	}
}

// LoadTestData loads test cases from a JSON file
func (l *TestLoader) LoadTestData(filename string) (*TestData, error) {
	filePath := filepath.Join(l.testDataPath, filename)
	
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var testData TestData
	if err := json.NewDecoder(file).Decode(&testData); err != nil {
		return nil, err
	}

	return &testData, nil
}

// CreateMockOrganizationFromSetup creates a MockOrganization from setup data
func CreateMockOrganizationFromSetup(setup MockOrganizationSetup) *MockOrganization {
	ownerID, _ := uuid.Parse(setup.OwnerID)
	now := time.Now()
	
	return &MockOrganization{
		ID:          uuid.New(),
		Name:        setup.Name,
		Description: setup.Description,
		OwnerID:     ownerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CreateMockOrganizationsFromSetup creates multiple MockOrganizations from setup data
func CreateMockOrganizationsFromSetup(setups []MockOrganizationSetup) []*MockOrganization {
	orgs := make([]*MockOrganization, len(setups))
	for i, setup := range setups {
		orgs[i] = CreateMockOrganizationFromSetup(setup)
	}
	return orgs
}

// GetStringValue safely extracts a string value from a map
func GetStringValue(data map[string]interface{}, key string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GetIntValue safely extracts an int value from a map
func GetIntValue(data map[string]interface{}, key string) int {
	if value, exists := data[key]; exists {
		if num, ok := value.(float64); ok {
			return int(num)
		}
	}
	return 0
}

// GetBoolValue safely extracts a bool value from a map
func GetBoolValue(data map[string]interface{}, key string) bool {
	if value, exists := data[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

// GetMapValue safely extracts a map value from a map
func GetMapValue(data map[string]interface{}, key string) map[string]interface{} {
	if value, exists := data[key]; exists {
		if m, ok := value.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// GetArrayValue safely extracts an array value from a map
func GetArrayValue(data map[string]interface{}, key string) []interface{} {
	if value, exists := data[key]; exists {
		if arr, ok := value.([]interface{}); ok {
			return arr
		}
	}
	return nil
} 