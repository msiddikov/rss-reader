package feed

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"rss-reader/internal/models"
	"strings"

	"github.com/hashicorp/go-uuid"
)

// ParseWithChatly sends the RawListing data to an external Chatly AI service,
// requesting it to generate a models.Job struct from the provided job information.
// The function constructs a prompt with the job data, sends it as a POST request,
// and expects a JSON string response that matches the models.Job structure.
//
// Parameters:
//
//	x RawListing - The job data to be parsed and transformed by Chatly.
//
// Returns:
//
//	(models.Job, error) - The parsed job struct on success, or an error if the process fails.
//
// Steps:
//  1. Generates a random chatId for the session.
//  2. Constructs a prompt string describing the desired output structure.
//  3. Sends the prompt to the Chatly backend as a JSON payload.
//  4. Reads and parses the JSON response, extracting the job data.
//  5. Unmarshals the job data into a models.Job struct and returns it.
//
// Errors:
//
//	Returns an error if any step fails, including HTTP request/response issues or JSON parsing errors.
func ParseWithChatly(x RawListing) (models.Job, error) {
	host := "https://chatly-back.lavina.tech"
	// generate random chatId
	chatId, _ := uuid.GenerateUUID()
	serviceId := "71259f43-b5d8-49df-9632-7c35ad278518"

	prompt := fmt.Sprintf("%s make from this information the following struct and response with parceable JSON string: {'company_id': <id>, 'title': <relevant title, if relevant then city-specific (e.g., Delivery Driver in New York) >, 'description': <Job description, if relevant then city-specific>, 'city': <job city>, 'country_code': <job country - ISO2>, 'source_job_id': <unique ID of job in source>, 'source': <source>, 'application_url': <job url in feed>, 'kind': <part-time/full-time/internship/traineeship>, 'proximity': <onsite/remote/hybrid>, 'work_visa_required': <boolean>, 'native_language_ids': <can be list of string of native language>, 'years_of_experience_lower': <min years of experience>, 'years_of_experience_upper': <max years of experience>, 'hours_per_week': <int>, 'duration': <3_months/6_months/12_months/indefinite>, 'paid_role': <boolean>, 'motivators': <list of motivators>, 'competencies': <list of competencies>, 'categories': <list of job categories>}", x)
	reqBody := struct {
		Message string `json:"message"`
	}{
		Message: prompt,
	}
	// Marshal the request body to string
	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return models.Job{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create a new HTTP request with the body
	req, err := http.NewRequest("POST", host+"/services/"+serviceId+"/chats/"+chatId+"/completion/", strings.NewReader(string(reqBodyJSON)))
	if err != nil {
		return models.Job{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.Job{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bd, _ := io.ReadAll(resp.Body)
		fmt.Println("Body:", string(bd))
		return models.Job{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var job models.Job

	bodybytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Job{}, fmt.Errorf("failed to read response body: %w", err)
	}

	response := struct {
		Data string `json:"data"`
	}{}

	err = json.Unmarshal([]byte(bodybytes), &response)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return models.Job{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	clean := strings.TrimPrefix(response.Data, "```json\n")
	clean = strings.TrimSuffix(clean, "\n```")

	err = json.Unmarshal([]byte(clean), &job)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return models.Job{}, fmt.Errorf("failed to unmarshal job data: %w", err)
	}

	return job, nil
}
