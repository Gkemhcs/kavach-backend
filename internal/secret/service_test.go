package secret

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
	"github.com/Gkemhcs/kavach-backend/internal/provider"
	providerdb "github.com/Gkemhcs/kavach-backend/internal/provider/gen"
	secretdb "github.com/Gkemhcs/kavach-backend/internal/secret/gen"
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
	Success             bool                     `json:"success"`
	Error               interface{}              `json:"error"`
	ErrorCode           string                   `json:"error_code,omitempty"`
	SecretVersion       interface{}              `json:"secret_version,omitempty"`
	SecretVersions      []map[string]interface{} `json:"secret_versions,omitempty"`
	SecretVersionDetail interface{}              `json:"secret_version_detail,omitempty"`
	SyncResponse        interface{}              `json:"sync_response,omitempty"`
}

// MockSetup represents the mock configuration for a test case
type MockSetup struct {
	SecretRepo      interface{} `json:"secret_repo,omitempty"` // Can be MockConfig or []MockConfig
	ProviderService MockConfig  `json:"provider_service,omitempty"`
}

// MockConfig represents a single mock configuration
type MockConfig struct {
	Method string                 `json:"method"`
	Return map[string]interface{} `json:"return"`
}

// MockProviderFactory is a mock implementation of the ProviderFactory interface
type MockProviderFactory struct {
	mock.Mock
}

// CreateProvider mocks the CreateProvider method
func (m *MockProviderFactory) CreateProvider(providerType provider.ProviderType, credentials map[string]interface{}, config map[string]interface{}) (provider.ProviderSyncer, error) {
	args := m.Called(providerType, credentials, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(provider.ProviderSyncer), args.Error(1)
}

// GetSupportedProviders mocks the GetSupportedProviders method
func (m *MockProviderFactory) GetSupportedProviders() []provider.ProviderType {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]provider.ProviderType)
}

// MockProviderSyncer is a mock implementation of the ProviderSyncer interface
type MockProviderSyncer struct {
	mock.Mock
}

// Sync mocks the Sync method
func (m *MockProviderSyncer) Sync(ctx context.Context, secrets []provider.Secret) ([]provider.SyncResult, error) {
	args := m.Called(ctx, secrets)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]provider.SyncResult), args.Error(1)
}

// ValidateCredentials mocks the ValidateCredentials method
func (m *MockProviderSyncer) ValidateCredentials(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// GetProviderName mocks the GetProviderName method
func (m *MockProviderSyncer) GetProviderName() string {
	args := m.Called()
	return args.String(0)
}

// SecretServiceTestSuite represents the test suite for SecretService
type SecretServiceTestSuite struct {
	suite.Suite
	service             *SecretService
	mockRepo            *MockSecretRepository
	mockProviderRepo    *provider.MockProviderRepository
	mockProviderFactory *MockProviderFactory
	encryptionService   *EncryptionService
	providerService     *provider.ProviderService
	logger              *logrus.Logger
	ctx                 context.Context
}

// SetupSuite sets up the test suite
func (suite *SecretServiceTestSuite) SetupSuite() {
	// Suppress logrus output during tests
	suite.logger = logrus.New()
	suite.logger.SetOutput(io.Discard)
}

// SetupTest sets up each individual test
func (suite *SecretServiceTestSuite) SetupTest() {
	suite.mockRepo = &MockSecretRepository{}
	suite.mockProviderRepo = &provider.MockProviderRepository{}
	suite.mockProviderFactory = &MockProviderFactory{}

	// Create a real encryption service for testing
	var err error
	suite.encryptionService, err = NewEncryptionService("w+1oDLMqyMZ1JEbTyS+2QOqaVZ/hnDfvIh/DuunlufA=", suite.logger) // base64 encoded 32-byte key
	require.NoError(suite.T(), err, "Failed to create encryption service")

	// Create a real encryptor for the provider service
	testEncryptor, err := utils.NewEncryptor("w+1oDLMqyMZ1JEbTyS+2QOqaVZ/hnDfvIh/DuunlufA=")
	require.NoError(suite.T(), err, "Failed to create encryptor")

	// Create provider service with mock repository
	suite.providerService = provider.NewProviderService(
		suite.mockProviderRepo,
		suite.mockProviderFactory,
		suite.logger,
		testEncryptor,
	)

	// Create secret service with real encryption and provider services
	suite.service = NewSecretService(
		suite.mockRepo,
		suite.encryptionService,
		suite.providerService,
		suite.logger,
	)

	suite.ctx = context.Background()
}

// TearDownTest cleans up after each test
func (suite *SecretServiceTestSuite) TearDownTest() {
	// Reset all mock expectations
	suite.mockRepo.ExpectedCalls = nil
	suite.mockProviderRepo.ExpectedCalls = nil
	suite.mockProviderFactory.ExpectedCalls = nil
}

// loadTestData loads test data from embedded JSON files
func (suite *SecretServiceTestSuite) loadTestData(filename string) *TestData {
	data, err := testDataFS.ReadFile("test_data/" + filename)
	require.NoError(suite.T(), err, "Failed to read test data file: %s", filename)

	var testData TestData
	err = json.Unmarshal(data, &testData)
	require.NoError(suite.T(), err, "Failed to unmarshal test data from: %s", filename)

	return &testData
}

// validateErrorCode validates that the error has the expected error code or message
func (suite *SecretServiceTestSuite) validateErrorCode(err error, expectedErrorCode string) {
	if expectedErrorCode == "" {
		return
	}

	// Try to cast to APIError to get the code
	if apiErr, ok := err.(*appErrors.APIError); ok {
		require.Equal(suite.T(), expectedErrorCode, apiErr.Code, "Error code mismatch")
	} else {
		// Fallback to checking if the error message contains the expected text
		require.Contains(suite.T(), err.Error(), expectedErrorCode, "Error message does not contain expected text")
	}
}

// TestCreateVersionWithData tests CreateVersion with data-driven test cases
func (suite *SecretServiceTestSuite) TestCreateVersionWithData() {
	testData := suite.loadTestData("create_secret_version_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupCreateVersionMocks(tc.MockSetup)

			// Build request from test input
			req := suite.buildCreateVersionRequest(tc.Input)
			environmentID := tc.Input["environment_id"].(string)

			// Call the service method
			result, err := suite.service.CreateVersion(suite.ctx, environmentID, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedVersion := tc.Expected.SecretVersion.(map[string]interface{})
				assert.Equal(suite.T(), expectedVersion["environment_id"].(string), result.EnvironmentID.String(), "Environment ID mismatch")
				assert.Equal(suite.T(), expectedVersion["commit_message"].(string), result.CommitMessage, "Commit message mismatch")
				assert.Equal(suite.T(), int(expectedVersion["secret_count"].(float64)), result.SecretCount, "Secret count mismatch")
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

// TestListVersionsWithData tests ListVersions with data-driven test cases
func (suite *SecretServiceTestSuite) TestListVersionsWithData() {
	testData := suite.loadTestData("list_versions_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupListVersionsMocks(tc.MockSetup)

			// Get input parameters
			environmentID := tc.Input["environment_id"].(string)

			// Call the service method
			result, err := suite.service.ListVersions(suite.ctx, environmentID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)

				// Validate result matches expected
				expectedVersions := tc.Expected.SecretVersions
				assert.Equal(suite.T(), len(expectedVersions), len(result), "Number of versions mismatch")

				for i, expectedVersion := range expectedVersions {
					if i < len(result) {
						assert.Equal(suite.T(), expectedVersion["commit_message"].(string), result[i].CommitMessage, "Commit message mismatch")
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

// TestGetVersionDetailsWithData tests GetVersionDetails with data-driven test cases
func (suite *SecretServiceTestSuite) TestGetVersionDetailsWithData() {
	testData := suite.loadTestData("get_version_details_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupGetVersionDetailsMocks(tc.MockSetup)

			// Get input parameters
			versionID := tc.Input["version_id"].(string)

			// Call the service method
			result, err := suite.service.GetVersionDetails(suite.ctx, versionID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedDetail := tc.Expected.SecretVersionDetail.(map[string]interface{})
				assert.Equal(suite.T(), expectedDetail["commit_message"].(string), result.CommitMessage, "Commit message mismatch")
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

// TestRollbackToVersionWithData tests RollbackToVersion with data-driven test cases
func (suite *SecretServiceTestSuite) TestRollbackToVersionWithData() {
	testData := suite.loadTestData("rollback_version_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupRollbackVersionMocks(tc.MockSetup)

			// Build request from test input
			req := suite.buildRollbackRequest(tc.Input)
			environmentID := tc.Input["environment_id"].(string)

			// Call the service method
			result, err := suite.service.RollbackToVersion(suite.ctx, environmentID, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedVersion := tc.Expected.SecretVersion.(map[string]interface{})
				assert.Equal(suite.T(), expectedVersion["commit_message"].(string), result.CommitMessage, "Commit message mismatch")
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

// TestSyncSecretsWithData tests SyncSecrets with data-driven test cases
func (suite *SecretServiceTestSuite) TestSyncSecretsWithData() {
	testData := suite.loadTestData("sync_secrets_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupSyncSecretsMocks(tc.MockSetup, tc)

			// Build request from test input
			req := suite.buildSyncSecretsRequest(tc.Input)
			environmentID := tc.Input["environment_id"].(string)

			// Call the service method
			result, err := suite.service.SyncSecrets(suite.ctx, environmentID, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedResponse := tc.Expected.SyncResponse.(map[string]interface{})
				assert.Equal(suite.T(), expectedResponse["provider"].(string), result.Provider, "Provider mismatch")
				assert.Equal(suite.T(), expectedResponse["status"].(string), result.Status, "Status mismatch")
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

// Setup functions for different test methods
func (suite *SecretServiceTestSuite) setupCreateVersionMocks(mockSetup MockSetup) {
	suite.setupSecretRepoMocks(mockSetup.SecretRepo)
}

func (suite *SecretServiceTestSuite) setupListVersionsMocks(mockSetup MockSetup) {
	suite.setupSecretRepoMocks(mockSetup.SecretRepo)
}

func (suite *SecretServiceTestSuite) setupGetVersionDetailsMocks(mockSetup MockSetup) {
	suite.setupSecretRepoMocks(mockSetup.SecretRepo)
}

func (suite *SecretServiceTestSuite) setupRollbackVersionMocks(mockSetup MockSetup) {
	suite.setupSecretRepoMocks(mockSetup.SecretRepo)
}

func (suite *SecretServiceTestSuite) setupSyncSecretsMocks(mockSetup MockSetup, tc TestCase) {
	suite.setupSecretRepoMocks(mockSetup.SecretRepo)
	if mockSetup.ProviderService.Method != "" {
		suite.setupProviderServiceMock(mockSetup.ProviderService)
		// Set up provider factory mock
		suite.setupProviderFactoryMock(mockSetup, tc)
	}
}

// setupSecretRepoMocks sets up multiple secret repository mocks
func (suite *SecretServiceTestSuite) setupSecretRepoMocks(secretRepo interface{}) {
	if secretRepo == nil {
		return
	}

	// Handle array of mock configs
	if configs, ok := secretRepo.([]interface{}); ok {
		for _, config := range configs {
			if mockConfig, ok := config.(map[string]interface{}); ok {
				suite.setupSecretRepoMockFromMap(mockConfig)
			}
		}
		return
	}

	// Handle single mock config
	if config, ok := secretRepo.(map[string]interface{}); ok {
		suite.setupSecretRepoMockFromMap(config)
		return
	}

	// Handle legacy MockConfig type
	if config, ok := secretRepo.(MockConfig); ok {
		suite.setupSecretRepoMock(config)
	}
}

// setupSecretRepoMockFromMap sets up a secret repository mock from a map
func (suite *SecretServiceTestSuite) setupSecretRepoMockFromMap(config map[string]interface{}) {
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
	suite.setupSecretRepoMock(mockConfig)
}

// setupSecretRepoMock sets up the secret repository mock
func (suite *SecretServiceTestSuite) setupSecretRepoMock(config MockConfig) {
	switch config.Method {
	case "CreateSecretVersion":
		if config.Return["error"] != nil {
			suite.mockRepo.On("CreateSecretVersion", suite.ctx, mock.AnythingOfType("secretdb.CreateSecretVersionParams")).
				Return(secretdb.SecretVersion{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock secret version from test data
			versionData := config.Return["secret_version"].(map[string]interface{})
			version := secretdb.SecretVersion{
				ID:            versionData["id"].(string),
				EnvironmentID: uuid.MustParse(versionData["environment_id"].(string)),
				CommitMessage: versionData["commit_message"].(string),
			}
			version.CreatedAt = time.Now()

			suite.mockRepo.On("CreateSecretVersion", suite.ctx, mock.AnythingOfType("secretdb.CreateSecretVersionParams")).
				Return(version, nil).Once()
		}
	case "InsertSecret":
		if config.Return["error"] != nil {
			suite.mockRepo.On("InsertSecret", suite.ctx, mock.AnythingOfType("secretdb.InsertSecretParams")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockRepo.On("InsertSecret", suite.ctx, mock.AnythingOfType("secretdb.InsertSecretParams")).
				Return(nil).Once()
		}
	case "ListSecretVersions":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListSecretVersions", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return([]secretdb.SecretVersion{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock secret versions from test data
			versions := []secretdb.SecretVersion{}
			if config.Return["secret_versions"] != nil {
				for _, v := range config.Return["secret_versions"].([]interface{}) {
					versionMap := v.(map[string]interface{})
					version := secretdb.SecretVersion{
						ID:            versionMap["id"].(string),
						EnvironmentID: uuid.MustParse(versionMap["environment_id"].(string)),
						CommitMessage: versionMap["commit_message"].(string),
					}
					version.CreatedAt = time.Now()
					versions = append(versions, version)
				}
			}

			suite.mockRepo.On("ListSecretVersions", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(versions, nil).Once()
		}
	case "GetSecretVersion":
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

			suite.mockRepo.On("GetSecretVersion", suite.ctx, mock.AnythingOfType("string")).
				Return(secretdb.SecretVersion{}, err).Once()
		} else {
			// Build mock secret version from test data
			versionData := config.Return["secret_version"].(map[string]interface{})
			version := secretdb.SecretVersion{
				ID:            versionData["id"].(string),
				EnvironmentID: uuid.MustParse(versionData["environment_id"].(string)),
				CommitMessage: versionData["commit_message"].(string),
			}
			version.CreatedAt = time.Now()

			suite.mockRepo.On("GetSecretVersion", suite.ctx, mock.AnythingOfType("string")).
				Return(version, nil).Once()
		}
	case "GetSecretsForVersion":
		if config.Return["error"] != nil {
			suite.mockRepo.On("GetSecretsForVersion", suite.ctx, mock.AnythingOfType("string")).
				Return([]secretdb.GetSecretsForVersionRow{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock secrets from test data
			secrets := []secretdb.GetSecretsForVersionRow{}
			if config.Return["secrets"] != nil {
				for _, s := range config.Return["secrets"].([]interface{}) {
					secretMap := s.(map[string]interface{})

					// Get the expected decrypted value from the test data
					expectedValue := secretMap["value_encrypted"].(string)

					// Handle special case for decryption error test
					var encryptedValue []byte
					var err error
					if expectedValue == "corrupted_encrypted_value" {
						// Use corrupted data for decryption error test
						encryptedValue = []byte("corrupted_data_that_will_fail_decryption")
					} else {
						// Encrypt the value using the real encryption service for testing
						encryptedValue, err = suite.encryptionService.Encrypt(expectedValue)
						if err != nil {
							// If encryption fails, use the original string as fallback
							encryptedValue = []byte(expectedValue)
						}
					}

					secret := secretdb.GetSecretsForVersionRow{
						ID:             uuid.MustParse(secretMap["id"].(string)),
						Name:           secretMap["name"].(string),
						ValueEncrypted: encryptedValue,
					}
					secrets = append(secrets, secret)
				}
			}

			suite.mockRepo.On("GetSecretsForVersion", suite.ctx, mock.AnythingOfType("string")).
				Return(secrets, nil).Once()
		}
	case "RollbackSecretsToVersion":
		if config.Return["error"] != nil {
			suite.mockRepo.On("RollbackSecretsToVersion", suite.ctx, mock.AnythingOfType("secretdb.RollbackSecretsToVersionParams")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockRepo.On("RollbackSecretsToVersion", suite.ctx, mock.AnythingOfType("secretdb.RollbackSecretsToVersionParams")).
				Return(nil).Once()
		}
	}
}

// setupProviderFactoryMock sets up the provider factory mock
func (suite *SecretServiceTestSuite) setupProviderFactoryMock(mockSetup MockSetup, tc TestCase) {
	// Create a mock provider syncer
	mockSyncer := &MockProviderSyncer{}

	// Check if this is a provider sync error test case
	if tc.Name == "provider_sync_error" {
		// Set up the mock syncer to return sync error
		mockSyncer.On("Sync", mock.Anything, mock.AnythingOfType("[]provider.Secret")).
			Return(nil, errors.New("sync operation failed")).Once()
	} else {
		// Set up the mock syncer to return successful sync results by default
		mockSyncer.On("Sync", mock.Anything, mock.AnythingOfType("[]provider.Secret")).
			Return([]provider.SyncResult{
				{Name: "DATABASE_URL", Success: true, Error: ""},
				{Name: "API_KEY", Success: true, Error: ""},
			}, nil).Once()
	}

	// Set up the factory to return the mock syncer
	suite.mockProviderFactory.On("CreateProvider", mock.AnythingOfType("provider.ProviderType"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("map[string]interface {}")).
		Return(mockSyncer, nil).Once()

	// Set up the factory to return supported providers
	suite.mockProviderFactory.On("GetSupportedProviders").
		Return([]provider.ProviderType{provider.ProviderGitHub, provider.ProviderGCP, provider.ProviderAzure}).Once()
}

// setupProviderServiceMock sets up the provider service mock
func (suite *SecretServiceTestSuite) setupProviderServiceMock(config MockConfig) {
	switch config.Method {
	case "GetProviderSyncer":
		if config.Return["error"] != nil {
			suite.mockProviderRepo.On("GetProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.GetProviderCredentialParams")).
				Return(providerdb.ProviderCredential{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Create test credentials and encrypt them
			testCredentials := map[string]interface{}{
				"token": "test_token_123",
			}
			credentialsJSON, err := json.Marshal(testCredentials)
			if err != nil {
				suite.T().Fatalf("Failed to marshal test credentials: %v", err)
			}

			// Use the test encryptor to encrypt the credentials
			testEncryptor, err := utils.NewEncryptor("w+1oDLMqyMZ1JEbTyS+2QOqaVZ/hnDfvIh/DuunlufA=")
			if err != nil {
				suite.T().Fatalf("Failed to create test encryptor: %v", err)
			}
			encryptedCredentials, err := testEncryptor.Encrypt(credentialsJSON)
			if err != nil {
				suite.T().Fatalf("Failed to encrypt test credentials: %v", err)
			}

			// Build mock provider credential
			credential := providerdb.ProviderCredential{
				ID:            uuid.New(),
				EnvironmentID: uuid.New(),
				Provider:      "github",
				Credentials:   json.RawMessage(encryptedCredentials),
				Config:        json.RawMessage(`{"owner": "testowner", "repository": "testrepo"}`),
			}
			credential.CreatedAt = time.Now()
			credential.UpdatedAt = time.Now()

			suite.mockProviderRepo.On("GetProviderCredential", suite.ctx, mock.AnythingOfType("providerdb.GetProviderCredentialParams")).
				Return(credential, nil).Once()
		}
	}
}

// Helper functions to build requests from test input
func (suite *SecretServiceTestSuite) buildCreateVersionRequest(input map[string]interface{}) CreateSecretVersionRequest {
	req := CreateSecretVersionRequest{
		CommitMessage: input["commit_message"].(string),
	}

	if input["secrets"] != nil {
		for _, s := range input["secrets"].([]interface{}) {
			secretMap := s.(map[string]interface{})
			secret := SecretInput{
				Name:  secretMap["name"].(string),
				Value: secretMap["value"].(string),
			}
			req.Secrets = append(req.Secrets, secret)
		}
	}

	return req
}

func (suite *SecretServiceTestSuite) buildRollbackRequest(input map[string]interface{}) RollbackRequest {
	return RollbackRequest{
		VersionID:     input["version_id"].(string),
		CommitMessage: input["commit_message"].(string),
	}
}

func (suite *SecretServiceTestSuite) buildSyncSecretsRequest(input map[string]interface{}) SyncSecretsRequest {
	req := SyncSecretsRequest{
		Provider: input["provider"].(string),
	}

	if input["version_id"] != nil {
		req.VersionID = input["version_id"].(string)
	}

	return req
}

// TestSecretServiceTestSuite runs the test suite
func TestSecretServiceTestSuite(t *testing.T) {
	suite.Run(t, new(SecretServiceTestSuite))
}
