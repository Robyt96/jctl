package client

import "strings"

// Job represents a Jenkins job/pipeline
type Job struct {
	Name        string        `json:"name"`
	URL         string        `json:"url"`
	Description string        `json:"description"`
	Color       string        `json:"color"`
	Buildable   bool          `json:"buildable"`
	Class       string        `json:"_class"` // Used to identify folders vs jobs
	LastBuild   *BuildRef     `json:"lastBuild,omitempty"`
	Builds      []Build       `json:"builds,omitempty"`
	Property    []JobProperty `json:"property,omitempty"`
}

// IsFolder returns true if this job is a folder
func (j *Job) IsFolder() bool {
	return strings.Contains(j.Class, "Folder") ||
		strings.Contains(j.Class, "folder") ||
		j.Class == "com.cloudbees.hudson.plugins.folder.Folder"
}

// GetParameterDefinitions extracts parameter definitions from job properties
func (j *Job) GetParameterDefinitions() []ParameterDefinition {
	for _, prop := range j.Property {
		if len(prop.ParameterDefinitions) > 0 {
			return prop.ParameterDefinitions
		}
	}
	return nil
}

// IsParameterized returns true if the job accepts parameters
func (j *Job) IsParameterized() bool {
	return len(j.GetParameterDefinitions()) > 0
}

// GetRequiredParameters returns a list of parameter names that don't have default values
func (j *Job) GetRequiredParameters() []string {
	var required []string
	for _, param := range j.GetParameterDefinitions() {
		// If there's no default value, the parameter is required
		if param.DefaultValue == nil || param.DefaultValue.Value == nil {
			required = append(required, param.Name)
		}
	}
	return required
}

// BuildRef is a reference to a build
type BuildRef struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// Build represents a Jenkins build
type Build struct {
	Number     int         `json:"number"`
	URL        string      `json:"url"`
	Result     string      `json:"result"` // SUCCESS, FAILURE, ABORTED, UNSTABLE, null (in progress)
	Timestamp  int64       `json:"timestamp"`
	Duration   int64       `json:"duration"`
	Building   bool        `json:"building"`
	Parameters []Parameter `json:"actions,omitempty"`
}

// Parameter represents a build parameter
type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// JobProperty represents job properties including parameters
type JobProperty struct {
	ParameterDefinitions []ParameterDefinition `json:"parameterDefinitions,omitempty"`
}

// ParameterDefinition defines a parameter for a job
type ParameterDefinition struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	DefaultValue *ParameterDefaultValue `json:"defaultParameterValue,omitempty"`
}

// ParameterDefaultValue represents the default value for a parameter
type ParameterDefaultValue struct {
	Value interface{} `json:"value"`
}

// QueueItem represents an item in the build queue
type QueueItem struct {
	ID         int            `json:"id"`
	Task       TaskRef        `json:"task"`
	Why        string         `json:"why"`
	Blocked    bool           `json:"blocked"`
	Buildable  bool           `json:"buildable"`
	Executable *ExecutableRef `json:"executable,omitempty"`
}

// TaskRef is a reference to a task
type TaskRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ExecutableRef is a reference to an executable (running build)
type ExecutableRef struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// JobsResponse represents the response from listing jobs
type JobsResponse struct {
	Jobs []Job `json:"jobs"`
}

// ProgressiveLogResponse represents a progressive log response from Jenkins
type ProgressiveLogResponse struct {
	Content     string // New log content since last request
	NextOffset  int64  // Byte offset for next request (from X-Text-Size header)
	HasMoreData bool   // Whether build is still running (from X-More-Data header)
}
