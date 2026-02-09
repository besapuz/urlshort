package router

// generate:reset
type TestConfig struct {
	Port        int
	Host        string
	Enabled     bool
	MaxRequests int
	Timeout     float64
}

// generate:reset
type UserData struct {
	ID       string
	Name     string
	Email    *string
	Roles    []string
	Settings map[string]interface{}
	Scores   []int
	Metadata map[string]string
}

// generate:reset
type ComplexStruct struct {
	Config  TestConfig
	Users   []*UserData
	Cache   map[string]*UserData
	Counter *int
	Logger  *Logger
	parent  *ComplexStruct // приватное поле
}

type Logger struct {
	level int
}
