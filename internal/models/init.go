package models

type Job struct {
	CompanyID              string   `json:"company_id"`
	Title                  string   `json:"title"`
	Description            string   `json:"description"`
	City                   string   `json:"city"`
	CountryCode            string   `json:"country_code"`
	SourceJobID            string   `json:"source_job_id"`
	Source                 string   `json:"source"`
	ApplicationURL         string   `json:"application_url"`
	Kind                   string   `json:"kind"`
	Proximity              string   `json:"proximity"`
	WorkVisaRequired       bool     `json:"work_visa_required"`
	NativeLanguageIDs      []string `json:"native_language_ids"`
	YearsOfExperienceLower int      `json:"years_of_experience_lower"`
	YearsOfExperienceUpper int      `json:"years_of_experience_upper"`
	HoursPerWeek           int      `json:"hours_per_week"`
	Duration               string   `json:"duration"`
	PaidRole               bool     `json:"paid_role"`
	Motivators             []string `json:"motivators"`
	Competencies           []string `json:"competencies"`
	Categories             []string `json:"categories"`
}
