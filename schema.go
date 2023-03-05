package main

type PipInspection struct {
	Version     string              `json:"version"`
	PipVersion  string              `json:"pip_version"`
	Installed   []InspectReportItem `json:"installed"`
	Environment Environment         `json:"environment"`
}

type Metadata struct {
	Author                 string   `json:"author"`
	AuthorEmail            string   `json:"author_email"`
	Classifier             []string `json:"classifier"`
	Description            string   `json:"description"`
	DescriptionContentType string   `json:"description_content_type"`
	DownloadURL            string   `json:"download_url"`
	HomePage               string   `json:"home_page"`
	Keywords               []string `json:"keywords"`
	License                string   `json:"license"`
	Maintainer             string   `json:"maintainer"`
	MaintainerEmail        string   `json:"maintainer_email"`
	MetadataVersion        string   `json:"metadata_version"`
	Name                   string   `json:"name"`
	Platform               []string `json:"platform"`
	ProjectURL             []string `json:"project_url"`
	ProvidesExtra          []string `json:"provides_extra"`
	RequiresDist           []string `json:"requires_dist"`
	RequiresPython         string   `json:"requires_python"`
	Summary                string   `json:"summary"`
	Version                string   `json:"version"`
}

type InspectReportItem struct {
	Metadata         Metadata `json:"metadata,omitempty"`
	MetadataLocation string   `json:"metadata_location"`
	Installer        string   `json:"installer"`
	Requested        bool     `json:"requested"`
}

type Environment struct {
	ImplementationName           string `json:"implementation_name"`
	ImplementationVersion        string `json:"implementation_version"`
	OsName                       string `json:"os_name"`
	PlatformMachine              string `json:"platform_machine"`
	PlatformRelease              string `json:"platform_release"`
	PlatformSystem               string `json:"platform_system"`
	PlatformVersion              string `json:"platform_version"`
	PythonFullVersion            string `json:"python_full_version"`
	PlatformPythonImplementation string `json:"platform_python_implementation"`
	PythonVersion                string `json:"python_version"`
	SysPlatform                  string `json:"sys_platform"`
}
