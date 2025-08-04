package provider

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	providerdb "github.com/Gkemhcs/kavach-backend/internal/provider/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//go:embed test_data/*.json
var testDataFS embed.FS

// TestData represents the structure of our test data files
type TestData struct {
	TestCases []TestCase `json:"test_cases"`
}

// TestCase represents a single test case
type TestCase struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Expected    ExpectedResult         `json:"expected"`
	MockSetup   MockSetup              `json:"mock_setup"`
}

// ExpectedResult represents the expected outcome of a test
type ExpectedResult struct {
	Success   bool        `json:"success"`
	Error     interface{} `json:"error"`
	ErrorCode string      `json:"error_code,omitempty"`
	// Additional fields for specific test types
	ProviderCredential  interface{}              `json:"provider_credential,omitempty"`
	ProviderCredentials []map[string]interface{} `json:"provider_credentials,omitempty"`
}

// MockSetup represents the mock configuration for a test case
type MockSetup struct {
	ProviderRepo interface{} `json:"provider_repo,omitempty"` // Can be MockConfig or []MockConfig
	Factory      MockConfig  `json:"factory,omitempty"`
	Encryptor    MockConfig  `json:"encryptor,omitempty"`
}

// MockConfig represents a single mock configuration
type MockConfig struct {
	Method string                 `json:"method"`
	Return map[string]interface{} `json:"return"`
}

// ProviderServiceTestSuite represents the test suite for ProviderService
type ProviderServiceTestSuite struct {
	suite.Suite
	service     *ProviderService
	mockRepo    *MockProviderRepository
	mockFactory *MockProviderFactory
	logger      *logrus.Logger
	ctx         context.Context
}

// MockProviderFactory is a mock implementation of the ProviderFactory interface
type MockProviderFactory struct {
	mock.Mock
}

// SetupSuite sets up the test suite
func (suite *ProviderServiceTestSuite) SetupSuite() {
	// Suppress logrus output during tests
	suite.logger = logrus.New()
	suite.logger.SetOutput(io.Discard)
}

// SetupTest sets up each individual test
func (suite *ProviderServiceTestSuite) SetupTest() {
	suite.mockRepo = &MockProviderRepository{}
	suite.mockFactory = &MockProviderFactory{}

	// Create a real encryptor for testing since it's a struct, not an interface
	testEncryptor, err := utils.NewEncryptor("w+1oDLMqyMZ1JEbTyS+2QOqaVZ/hnDfvIh/DuunlufA=") // base64 encoded 32-byte key
	require.NoError(suite.T(), err, "Failed to create encryptor")
	suite.service = NewProviderService(
		suite.mockRepo,
		suite.mockFactory,
		suite.logger,
		testEncryptor,
	)

	suite.ctx = context.Background()
}

// TearDownTest cleans up after each test
func (suite *ProviderServiceTestSuite) TearDownTest() {
	// Reset all mock expectations
	suite.mockRepo.ExpectedCalls = nil
	suite.mockFactory.ExpectedCalls = nil
}

// loadTestData loads test data from embedded JSON files
func (suite *ProviderServiceTestSuite) loadTestData(filename string) *TestData {
	data, err := testDataFS.ReadFile("test_data/" + filename)
	require.NoError(suite.T(), err, "Failed to read test data file: %s", filename)

	var testData TestData
	err = json.Unmarshal(data, &testData)
	require.NoError(suite.T(), err, "Failed to unmarshal test data from: %s", filename)

	return &testData
}

// validateErrorCode validates that the error has the expected error code or message
func (suite *ProviderServiceTestSuite) validateErrorCode(err error, expectedErrorCode string) {
	if expectedErrorCode == "" {
		return
	}

	// Try to cast to APIError to get the code
	if apiErr, ok := err.(*appErrors.APIError); ok {
		require.Equal(suite.T(), expectedErrorCode, apiErr.Code, "Error code mismatch")
	} else {
		// Fallback to checking if the error message contains the expected text
		// This is for cases where the error might be wrapped or is a plain error
		require.Contains(suite.T(), err.Error(), expectedErrorCode, "Error message does not contain expected text")
	}
}

// TestCreateProviderCredentialWithData tests CreateProviderCredential with data-driven test cases
func (suite *ProviderServiceTestSuite) TestCreateProviderCredentialWithData() {
	testData := suite.loadTestData("create_provider_credential_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupCreateProviderCredentialMocks(tc.MockSetup)

			// Build request from test input
			req := suite.buildCreateProviderCredentialRequest(tc.Input)
			environmentID := tc.Input["environment_id"].(string)
			userID := tc.Input["user_id"].(string)

			// Call the service method
			result, err := suite.service.CreateProviderCredential(suite.ctx, environmentID, userID, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedCredential := tc.Expected.ProviderCredential.(map[string]interface{})
				assert.Equal(suite.T(), expectedCredential["provider"].(string), string(result.Provider), "Provider mismatch")
				assert.Equal(suite.T(), expectedCredential["environment_id"].(string), result.EnvironmentID.String(), "Environment ID mismatch")
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
				// Validate that downstream methods were NOT called in error scenarios
				suite.validateNegativeMocks(tc.Name)
			}
		})
	}
}

// TestGetProviderCredentialWithData tests GetProviderCredential with data-driven test cases
func (suite *ProviderServiceTestSuite) TestGetProviderCredentialWithData() {
	testData := suite.loadTestData("get_provider_credential_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupGetProviderCredentialMocks(tc.MockSetup)

			// Get input parameters
			environmentID := tc.Input["environment_id"].(string)
			provider := tc.Input["provider"].(string)

			// Call the service method
			result, err := suite.service.GetProviderCredential(suite.ctx, environmentID, provider)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedCredential := tc.Expected.ProviderCredential.(map[string]interface{})
				assert.Equal(suite.T(), expectedCredential["provider"].(string), string(result.Provider), "Provider mismatch")
				assert.Equal(suite.T(), expectedCredential["environment_id"].(string), result.EnvironmentID.String(), "Environment ID mismatch")
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestListProviderCredentialsWithData tests ListProviderCredentials with data-driven test cases
func (suite *ProviderServiceTestSuite) TestListProviderCredentialsWithData() {
	testData := suite.loadTestData("list_provider_credentials_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupListProviderCredentialsMocks(tc.MockSetup)

			// Get input parameters
			environmentID := tc.Input["environment_id"].(string)

			// Call the service method
			result, err := suite.service.ListProviderCredentials(suite.ctx, environmentID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				// For ListProviderCredentials, an empty slice is valid, so we don't require NotNil
				// require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedCredentials := tc.Expected.ProviderCredentials
				assert.Equal(suite.T(), len(expectedCredentials), len(result), "Number of provider credentials mismatch")

				for i, expectedCredential := range expectedCredentials {
					if i < len(result) {
						assert.Equal(suite.T(), expectedCredential["provider"].(string), string(result[i].Provider), "Provider mismatch")
					}
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestUpdateProviderCredentialWithData tests UpdateProviderCredential with data-driven test cases
func (suite *ProviderServiceTestSuite) TestUpdateProviderCredentialWithData() {
	testData := suite.loadTestData("update_provider_credential_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupUpdateProviderCredentialMocks(tc.MockSetup)

			// Build request from test input
			req := suite.buildUpdateProviderCredentialRequest(tc.Input)
			environmentID := tc.Input["environment_id"].(string)
			provider := tc.Input["provider"].(string)

			// Call the service method
			result, err := suite.service.UpdateProviderCredential(suite.ctx, environmentID, provider, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedCredential := tc.Expected.ProviderCredential.(map[string]interface{})
				assert.Equal(suite.T(), expectedCredential["provider"].(string), string(result.Provider), "Provider mismatch")
				assert.Equal(suite.T(), expectedCredential["environment_id"].(string), result.EnvironmentID.String(), "Environment ID mismatch")
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
				// Validate that downstream methods were NOT called in error scenarios
				suite.validateNegativeMocks(tc.Name)
			}
		})
	}
}

// TestDeleteProviderCredentialWithData tests DeleteProviderCredential with data-driven test cases
func (suite *ProviderServiceTestSuite) TestDeleteProviderCredentialWithData() {
	testData := suite.loadTestData("delete_provider_credential_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupDeleteProviderCredentialMocks(tc.MockSetup)

			// Get input parameters
			environmentID := tc.Input["environment_id"].(string)
			provider := tc.Input["provider"].(string)

			// Call the service method
			err := suite.service.DeleteProviderCredential(suite.ctx, environmentID, provider)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestGetProviderSyncerWithData tests GetProviderSyncer with data-driven test cases
func (suite *ProviderServiceTestSuite) TestGetProviderSyncerWithData() {
	testData := suite.loadTestData("get_provider_syncer_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupGetProviderSyncerMocks(tc.MockSetup)

			// Get input parameters
			environmentID := tc.Input["environment_id"].(string)
			provider := tc.Input["provider"].(string)

			// Call the service method
			result, err := suite.service.GetProviderSyncer(suite.ctx, environmentID, provider)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// Mock implementations for the factory interface
func (m *MockProviderFactory) CreateProvider(providerType ProviderType, credentials map[string]interface{}, config map[string]interface{}) (ProviderSyncer, error) {
	args := m.Called(providerType, credentials, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ProviderSyncer), args.Error(1)
}

func (m *MockProviderFactory) GetSupportedProviders() []ProviderType {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]ProviderType)
}

// Setup functions for different test methods
func (suite *ProviderServiceTestSuite) setupCreateProviderCredentialMocks(mockSetup MockSetup) {
	suite.setupProviderRepoMocks(mockSetup.ProviderRepo)
	if mockSetup.Factory.Method != "" {
		suite.setupFactoryMock(mockSetup.Factory)
	}
}

func (suite *ProviderServiceTestSuite) setupGetProviderCredentialMocks(mockSetup MockSetup) {
	suite.setupProviderRepoMocks(mockSetup.ProviderRepo)
}

func (suite *ProviderServiceTestSuite) setupListProviderCredentialsMocks(mockSetup MockSetup) {
	suite.setupProviderRepoMocks(mockSetup.ProviderRepo)
}

func (suite *ProviderServiceTestSuite) setupUpdateProviderCredentialMocks(mockSetup MockSetup) {
	suite.setupProviderRepoMocks(mockSetup.ProviderRepo)
	if mockSetup.Factory.Method != "" {
		suite.setupFactoryMock(mockSetup.Factory)
	}
}

func (suite *ProviderServiceTestSuite) setupDeleteProviderCredentialMocks(mockSetup MockSetup) {
	suite.setupProviderRepoMocks(mockSetup.ProviderRepo)
}

func (suite *ProviderServiceTestSuite) setupGetProviderSyncerMocks(mockSetup MockSetup) {
	suite.setupProviderRepoMocks(mockSetup.ProviderRepo)
	if mockSetup.Factory.Method != "" {
		suite.setupFactoryMock(mockSetup.Factory)
	}
}

// validateNegativeMocks validates that certain methods were NOT called when they shouldn't be
func (suite *ProviderServiceTestSuite) validateNegativeMocks(testCase string) {
	// CreateProviderCredential error cases - these should NOT call downstream methods
	if strings.Contains(testCase, "invalid_provider_type") {
		suite.mockRepo.AssertNotCalled(suite.T(), "CreateProviderCredential")
	}

	if strings.Contains(testCase, "invalid_provider_data") {
		suite.mockRepo.AssertNotCalled(suite.T(), "CreateProviderCredential")
	}

	// UpdateProviderCredential error cases
	if strings.Contains(testCase, "invalid_provider_type") && strings.Contains(testCase, "update") {
		suite.mockRepo.AssertNotCalled(suite.T(), "UpdateProviderCredential")
	}

	if strings.Contains(testCase, "invalid_provider_data") && strings.Contains(testCase, "update") {
		suite.mockRepo.AssertNotCalled(suite.T(), "UpdateProviderCredential")
	}
}

// setupProviderRepoMocks sets up multiple provider repository mocks
func (suite *ProviderServiceTestSuite) setupProviderRepoMocks(providerRepo interface{}) {
	if providerRepo == nil {
		return
	}

	// Handle array of mock configs
	if configs, ok := providerRepo.([]interface{}); ok {
		for _, config := range configs {
			if mockConfig, ok := config.(map[string]interface{}); ok {
				suite.setupProviderRepoMockFromMap(mockConfig)
			}
		}
		return
	}

	// Handle single mock config
	if config, ok := providerRepo.(map[string]interface{}); ok {
		suite.setupProviderRepoMockFromMap(config)
		return
	}

	// Handle legacy MockConfig type
	if config, ok := providerRepo.(MockConfig); ok {
		suite.setupProviderRepoMock(config)
	}
}

// setupProviderRepoMockFromMap sets up a provider repository mock from a map
func (suite *ProviderServiceTestSuite) setupProviderRepoMockFromMap(config map[string]interface{}) {
	method, ok := config["method"].(string)
	if !ok {
		return
	}

	returnData, ok := config["return"].(map[string]interface{})
	if !ok {
		return
	}

	// Convert map back to MockConfig for existing logic
	mockConfig := MockConfig{
		Method: method,
		Return: returnData,
	}
	suite.setupProviderRepoMock(mockConfig)
}

// setupProviderRepoMock sets up the provider repository mock
func (suite *ProviderServiceTestSuite) setupProviderRepoMock(config MockConfig) {
	switch config.Method {
	case "CreateProviderCredential":
		if config.Return["error"] != nil {
			suite.mockRepo.On("CreateProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.CreateProviderCredentialParams")).
				Return(providerdb.ProviderCredential{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock provider credential from test data
			credentialData := config.Return["provider_credential"].(map[string]interface{})
			credential := providerdb.ProviderCredential{
				ID:            uuid.MustParse(credentialData["id"].(string)),
				EnvironmentID: uuid.MustParse(credentialData["environment_id"].(string)),
				Provider:      credentialData["provider"].(string),
				Config:        json.RawMessage(`{"test": "config"}`),
			}
			credential.CreatedAt = time.Now()
			credential.UpdatedAt = time.Now()

			suite.mockRepo.On("CreateProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.CreateProviderCredentialParams")).
				Return(credential, nil).Once()
		}
	case "GetProviderCredential":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			var err error

			// Handle specific error types
			switch errorMsg {
			case "sql: no rows in result set":
				err = sql.ErrNoRows
			default:
				err = errors.New(errorMsg)
			}

			suite.mockRepo.On("GetProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.GetProviderCredentialParams")).
				Return(providerdb.ProviderCredential{}, err).Once()
		} else {
			// Build mock provider credential from test data
			credentialData := config.Return["provider_credential"].(map[string]interface{})
			credential := providerdb.ProviderCredential{
				ID:            uuid.MustParse(credentialData["id"].(string)),
				EnvironmentID: uuid.MustParse(credentialData["environment_id"].(string)),
				Provider:      credentialData["provider"].(string),
				Config:        json.RawMessage(`{"test": "config"}`),
			}
			credential.CreatedAt = time.Now()
			credential.UpdatedAt = time.Now()

			suite.mockRepo.On("GetProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.GetProviderCredentialParams")).
				Return(credential, nil).Once()
		}
	case "ListProviderCredentials":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListProviderCredentials", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return([]providerdb.ProviderCredential{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock provider credentials from test data
			credentials := []providerdb.ProviderCredential{}
			if config.Return["provider_credentials"] != nil {
				for _, cred := range config.Return["provider_credentials"].([]interface{}) {
					credMap := cred.(map[string]interface{})
					credential := providerdb.ProviderCredential{
						ID:            uuid.MustParse(credMap["id"].(string)),
						EnvironmentID: uuid.MustParse(credMap["environment_id"].(string)),
						Provider:      credMap["provider"].(string),
						Config:        json.RawMessage(`{"test": "config"}`),
					}
					credential.CreatedAt = time.Now()
					credential.UpdatedAt = time.Now()
					credentials = append(credentials, credential)
				}
			}

			suite.mockRepo.On("ListProviderCredentials", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(credentials, nil).Once()
		}
	case "UpdateProviderCredential":
		if config.Return["error"] != nil {
			suite.mockRepo.On("UpdateProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.UpdateProviderCredentialParams")).
				Return(providerdb.ProviderCredential{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock provider credential from test data
			credentialData := config.Return["provider_credential"].(map[string]interface{})
			credential := providerdb.ProviderCredential{
				ID:            uuid.MustParse(credentialData["id"].(string)),
				EnvironmentID: uuid.MustParse(credentialData["environment_id"].(string)),
				Provider:      credentialData["provider"].(string),
				Config:        json.RawMessage(`{"test": "config"}`),
			}
			credential.CreatedAt = time.Now()
			credential.UpdatedAt = time.Now()

			suite.mockRepo.On("UpdateProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.UpdateProviderCredentialParams")).
				Return(credential, nil).Once()
		}
	case "DeleteProviderCredential":
		if config.Return["error"] != nil {
			suite.mockRepo.On("DeleteProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.DeleteProviderCredentialParams")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockRepo.On("DeleteProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.DeleteProviderCredentialParams")).
				Return(nil).Once()
		}
	}
}

// setupFactoryMock sets up the factory mock
func (suite *ProviderServiceTestSuite) setupFactoryMock(config MockConfig) {
	switch config.Method {
	case "GetSupportedProviders":
		if config.Return["providers"] != nil {
			providers := []ProviderType{}
			for _, p := range config.Return["providers"].([]interface{}) {
				providers = append(providers, ProviderType(p.(string)))
			}
			suite.mockFactory.On("GetSupportedProviders").Return(providers).Once()
		} else {
			suite.mockFactory.On("GetSupportedProviders").Return([]ProviderType{}).Once()
		}
	case "CreateProvider":
		if config.Return["error"] != nil {
			suite.mockFactory.On("CreateProvider", mock.AnythingOfType("ProviderType"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("map[string]interface {}")).
				Return(nil, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Create a mock provider syncer
			mockSyncer := &MockProviderSyncer{}
			suite.mockFactory.On("CreateProvider", mock.AnythingOfType("ProviderType"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("map[string]interface {}")).
				Return(mockSyncer, nil).Once()
		}
	}
}

// Helper functions to build requests from test input
func (suite *ProviderServiceTestSuite) buildCreateProviderCredentialRequest(input map[string]interface{}) CreateProviderCredentialRequest {
	req := CreateProviderCredentialRequest{
		Provider: ProviderType(input["provider"].(string)),
	}

	if input["credentials"] != nil {
		req.Credentials = input["credentials"].(map[string]interface{})
	}
	if input["config"] != nil {
		req.Config = input["config"].(map[string]interface{})
	}

	return req
}

func (suite *ProviderServiceTestSuite) buildUpdateProviderCredentialRequest(input map[string]interface{}) UpdateProviderCredentialRequest {
	req := UpdateProviderCredentialRequest{}

	if input["credentials"] != nil {
		req.Credentials = input["credentials"].(map[string]interface{})
	}
	if input["config"] != nil {
		req.Config = input["config"].(map[string]interface{})
	}

	return req
}

// MockProviderSyncer is a mock implementation of ProviderSyncer for testing
type MockProviderSyncer struct {
	mock.Mock
}

func (m *MockProviderSyncer) Sync(ctx context.Context, secrets []Secret) ([]SyncResult, error) {
	args := m.Called(ctx, secrets)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]SyncResult), args.Error(1)
}

func (m *MockProviderSyncer) ValidateCredentials(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockProviderSyncer) GetProviderName() string {
	args := m.Called()
	return args.String(0)
}

// TestProviderServiceTestSuite runs the test suite
func TestProviderServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderServiceTestSuite))
}
