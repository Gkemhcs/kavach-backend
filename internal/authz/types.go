package authz

type AdapterConfig struct {
	DB_HOST         string `json:"db_host" yaml:"db_host"`
	DB_PORT         string `json:"db_port" yaml:"db_port"`
	DB_USER         string `json:"db_user" yaml:"db_user"`
	DB_PASSWORD     string `json:"db_password" yaml:"db_password"`
	DB_NAME         string `json:"db_name" yaml:"db_name"`
	MODEL_FILE_PATH string `json:"model_file_path" yaml:"model_file_path"`
}
