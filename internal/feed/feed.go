package feed

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"rss-reader/internal/models"
	"strconv"
	"strings"
)

type RawListing struct {
	root map[string]any
}

var (
	useChatly  = true          // Set to true if you want to use Chatly for parsing
	outputFile = "output.json" // File to write the output, if needed
)

// ParseFeed fetches an RSS or Atom feed from the specified URL, checks for errors,
// and parses the XML content. The parsed output is written to a file specified by
// the global variable outputFile. It returns an error if fetching, file creation,
// or XML parsing fails.
func ParseFeed(feedURL string) error {
	// Requesting the feed, but not reading the body directly
	resp, err := http.Get(feedURL)
	if err != nil {
		log.Fatalf("Error fetching feed: %s", err)
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Error fetching feed: %s, Status Code: %d, Body: %s", feedURL, resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("error fetching feed: %s, status code: %d", feedURL, resp.StatusCode)
	}
	// Parse the XML from the response body
	log.Printf("Successfully fetched feed: %s", feedURL)

	// Prepare the output writer
	var outputWriter io.Writer
	file, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Error creating output file: %s", err)
		return err
	}
	defer file.Close()
	outputWriter = file

	err = ParseXML(resp.Body, outputWriter)
	if err != nil {
		log.Fatalf("Error parsing XML: %s", err)
		return err
	}

	return nil

}

// ParseXML reads an XML feed from the provided reader, extracts <job> elements,
// parses each job into a models.Job struct (using either ParseToJob or ParseWithChatly),
// and writes the resulting jobs as a JSON array to the provided writer.
//
// Parameters:
//
//	reader io.Reader - The XML input source.
//	writer io.Writer - The output destination for the JSON array.
//
// Behavior:
//   - Iterates through the XML tokens, looking for <job> elements.
//   - For each <job>, parses it into a RawListing and then into a models.Job.
//   - If useChatly is true, uses the Chatly AI service for parsing (up to 3 jobs for testing).
//   - Each parsed job is written to the output as a JSON object, separated by commas.
//   - The output is a valid JSON array.
//
// Returns:
//
//	error - If any error occurs during XML parsing, job conversion, or writing output.
//
// Notes:
//   - If writer is nil, returns an error.
//   - Logs errors for individual jobs but continues processing the rest.
func ParseXML(reader io.Reader, writer io.Writer) error {
	decoder := xml.NewDecoder(reader)
	count := 0

	if writer == nil {
		// If no writer is provided, we will just log the output
		return fmt.Errorf("no writer provided for output")
	}

	// start the output with [

	if _, err := writer.Write([]byte("[")); err != nil {
		return fmt.Errorf("failed to write start of JSON array: %w", err)
	}

	for {
		// Read tokens from the XML document in a stream.
		t, err := decoder.Token()

		// If we are at the end of the file, we are done
		if err == io.EOF {
			log.Println("The end")
			break
		} else if err != nil {
			log.Fatalf("Error decoding token: %s", err)
		} else if t == nil {
			break
		}

		// Here, we inspect the token
		switch se := t.(type) {

		// We have the start of an element.
		// However, we have the complete token in t
		case xml.StartElement:
			switch se.Name.Local {

			// Found a job, so we process it
			case "job":

				root := &RawListing{}
				root.UnmarshalXML(decoder, se)

				job := models.Job{}

				if useChatly {
					count++
					if count > 3 { // Limit to 3 jobs for testing
						log.Println("Limiting to 3 jobs for testing with Chatly")
						useChatly = false
					}

					job, err = ParseWithChatly(*root)
					if err != nil {
						log.Printf("Error parsing job: %s", err)
						continue
					}
				} else {
					job, err = root.ParseToJob()
					if err != nil {
						log.Printf("Error parsing job: %s", err)
						continue
					}
				}

				err = PushJobIntoFile(job, writer, count)
				if err != nil {
					log.Printf("Error writing job to output: %s", err)
					continue
				}

				// And use it for whatever we want to
				log.Printf("'%s' found", root.root["title"].(string))
			}
		}
	}
	// End the output with ]
	if _, err := writer.Write([]byte("\n]")); err != nil {
		log.Printf("Error writing end of JSON array: %s", err)
		return err
	}

	return nil
}

func (x *RawListing) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	x.root = map[string]any{"_": start.Name.Local}
	path := []map[string]any{x.root}
	for {

		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch elem := token.(type) {
		case xml.StartElement:
			newMap := map[string]any{"_": elem.Name.Local}
			path[len(path)-1][elem.Name.Local] = newMap
			path = append(path, newMap)
		case xml.EndElement:
			path = path[:len(path)-1]
			// If we reach the root element, we stop processing
			if elem.Name.Local == start.Name.Local {
				return nil
			}
		case xml.CharData:
			val := strings.TrimSpace(string(elem))
			if val == "" {
				break
			}
			curName := path[len(path)-1]["_"].(string)
			path[len(path)-2][curName] = typeConvert(val)
		}
	}
}

func typeConvert(s string) any {
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f
	}
	return s
}

// This function trims the values up to 1000 characters
func (x *RawListing) Trim() {
	for k, v := range x.root {
		if str, ok := v.(string); ok && len(str) > 1000 {
			x.root[k] = str[:1000]
		} else if arr, ok := v.([]any); ok {
			for i, val := range arr {
				if str, ok := val.(string); ok && len(str) > 1000 {
					arr[i] = str[:1000]
				}
			}
			x.root[k] = arr
		}
	}
}

func (x *RawListing) ParseToJob() (models.Job, error) {
	job := models.Job{}

	// Required fields (guaranteed to exist)
	if title, ok := x.root["title"].(string); ok {
		job.Title = title
	} else {
		return job, fmt.Errorf("title not found or not a string")
	}
	if description, ok := x.root["description"].(string); ok {
		job.Description = description
	}
	if applicationURL, ok := x.root["url"].(string); ok { // url maps to ApplicationURL
		job.ApplicationURL = applicationURL
	}
	if sourceJobID, ok := x.root["id"].(string); ok { // id maps to SourceJobID
		job.SourceJobID = sourceJobID
	}
	if city, ok := x.root["city"].(string); ok {
		job.City = city
	}
	if country, ok := x.root["country"].(string); ok { // country maps to CountryCode
		job.CountryCode = country
	}
	if company, ok := x.root["company"].(string); ok { // company maps to CompanyID
		job.CompanyID = company
	}

	// Optional fields (map to existing struct fields if possible)
	if currency, ok := x.root["currency"].(string); ok {
		job.Description += "\nCurrency: " + currency // Appending currency info to description
	}
	if postalcode, ok := x.root["postalcode"].(string); ok {
		job.Description += "\nPostal Code: " + postalcode
	}
	if region, ok := x.root["region"].(string); ok {
		job.Description += "\nRegion: " + region
	}

	return job, nil
}

func PushJobIntoFile(job models.Job, writer io.Writer, count int) error {
	// Marshal the job to JSON and write it to the output
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job to JSON: %w", err)
	}
	// Write a comma before the job if it's not the first job
	if count > 0 {
		if _, err := writer.Write([]byte(",\n")); err != nil {
			return fmt.Errorf("failed to write comma to output: %w", err)
		}
	}
	// Write the job JSON to the output
	if _, err := writer.Write(jobJSON); err != nil {
		return fmt.Errorf("failed to write job JSON to output: %w", err)
	}

	return nil
}
