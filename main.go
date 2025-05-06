package main

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
	probing "github.com/prometheus-community/pro-bing"
)

type Website struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Load the websites.yaml file
const fileName = "websites.yaml"

// WebsitesFile represents the structure of the websites.yaml file
type WebsitesFile struct {
	Websites []Website `yaml:"websites"`
}

func loadWebsitesFile() []Website {
	// Read the file
	fileHandle, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}

	// Unmarshal the file
	var websitesFile WebsitesFile
	err = yaml.NewDecoder(fileHandle).Decode(&websitesFile)
	if err != nil {
		panic(err)
	}

	return websitesFile.Websites
}

// PingResult stores the result of a ping operation
type PingResult struct {
	URL         string
	Domain      string
	PacketsSent int
	PacketsRecv int
	PacketLoss  float64
	AvgRtt      time.Duration
	Error       error
}

// FetchResult stores the result of a fetch operation
type FetchResult struct {
	URL        string
	StatusCode int
	BodyLength int
	BodySize   float64
	Error      error
	Redirects  []string
}

// TUI Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#7D56F4")).
			PaddingLeft(1).
			PaddingRight(1)

	cellStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E74C3C"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12"))

	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			MarginTop(1).
			MarginBottom(1)
)

func main() {
	// Clear the terminal
	fmt.Print("\033[H\033[2J")

	// Print app title
	appTitle := titleStyle.Render(" Async Web Data Dashboard ")
	fmt.Println(lipgloss.NewStyle().Width(80).Align(lipgloss.Center).Render(appTitle))
	fmt.Println()

	// Load the websites
	urls := loadWebsitesFile()

	// Start the timer
	start := time.Now()

	// Show loading spinner
	fmt.Println(infoStyle.Render(" ⏳ Pinging URLs..."))

	// Channel for ping results
	pingResults := make(chan PingResult, len(urls))

	// First ping all the urls
	for _, url := range urls {
		go pingUrl(url.URL, pingResults)
	}

	// Collect all ping results
	allPingResults := make([]PingResult, 0, len(urls))
	for i := 0; i < len(urls); i++ {
		result := <-pingResults
		allPingResults = append(allPingResults, result)
	}

	// Sort ping results by average time (descending)
	sort.Slice(allPingResults, func(i, j int) bool {
		// Handle errors (put errors at the end)
		if allPingResults[i].Error != nil {
			return false
		}
		if allPingResults[j].Error != nil {
			return true
		}
		// Sort by AvgRtt in descending order
		return allPingResults[i].AvgRtt > allPingResults[j].AvgRtt
	})

	// End the timer for pinging the urls
	pingTime := time.Since(start)

	// Start the timer for fetching the data
	start = time.Now()

	// Show loading spinner
	fmt.Println(infoStyle.Render(" ⏳ Fetching URL content..."))

	// Channel for fetch results
	fetchResults := make(chan FetchResult, len(urls))

	// Now fetch the data from all the urls
	for _, url := range urls {
		go fetchData(url.URL, fetchResults)
	}

	// Collect all fetch results
	allFetchResults := make([]FetchResult, 0, len(urls))
	for i := 0; i < len(urls); i++ {
		result := <-fetchResults
		allFetchResults = append(allFetchResults, result)
	}

	// Sort fetch results by body size (descending)
	sort.Slice(allFetchResults, func(i, j int) bool {
		// Handle errors (put errors at the end)
		if allFetchResults[i].Error != nil {
			return false
		}
		if allFetchResults[j].Error != nil {
			return true
		}
		// Sort by BodySize in descending order
		return allFetchResults[i].BodySize > allFetchResults[j].BodySize
	})

	// End the timer for fetching the data
	fetchTime := time.Since(start)

	// Display timing information
	timingTitle := titleStyle.Render(" Timing Information ")
	fmt.Println(lipgloss.NewStyle().Width(80).Align(lipgloss.Center).Render(timingTitle))

	// Properly align the timing table headers and values
	operationHeader := headerStyle.Width(40).Render("Operation")
	timeHeader := headerStyle.Width(40).Render("Time")
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, operationHeader, timeHeader)

	pingRow := lipgloss.JoinHorizontal(lipgloss.Top,
		cellStyle.Width(40).Render("Ping All URLs"),
		cellStyle.Width(40).Render(pingTime.String()),
	)

	fetchRow := lipgloss.JoinHorizontal(lipgloss.Top,
		cellStyle.Width(40).Render("Fetch All URLs"),
		cellStyle.Width(40).Render(fetchTime.String()),
	)

	timingTable := lipgloss.JoinVertical(lipgloss.Left,
		headerRow,
		pingRow,
		fetchRow,
	)

	fmt.Println(tableStyle.Width(80).Render(timingTable))

	// Print ping results table
	pingTitle := titleStyle.Render(" Ping Results ")
	fmt.Println(lipgloss.NewStyle().Width(80).Align(lipgloss.Center).Render(pingTitle))

	// Create ping table header
	pingTableHeader := []string{
		headerStyle.Width(30).Render("URL"),
		headerStyle.Width(10).Render("Sent"),
		headerStyle.Width(10).Render("Received"),
		headerStyle.Width(10).Render("Loss %"),
		headerStyle.Width(18).Render("Avg Time"),
	}

	pingHeaderRow := lipgloss.JoinHorizontal(lipgloss.Top, pingTableHeader...)

	// Create ping table rows
	var pingRows []string
	pingRows = append(pingRows, pingHeaderRow)

	for _, result := range allPingResults {
		var row string
		if result.Error != nil {
			row = lipgloss.JoinHorizontal(lipgloss.Top,
				cellStyle.Width(30).Render(truncateString(result.URL, 27)),
				errorStyle.Width(48).Render(fmt.Sprintf("Error: %v", result.Error)),
			)
		} else {
			recvStyle := cellStyle
			if result.PacketsRecv == 0 {
				recvStyle = errorStyle
			} else if result.PacketsRecv < result.PacketsSent {
				recvStyle = warningStyle
			} else {
				recvStyle = successStyle
			}

			lossStyle := cellStyle
			if result.PacketLoss > 50 {
				lossStyle = errorStyle
			} else if result.PacketLoss > 0 {
				lossStyle = warningStyle
			} else {
				lossStyle = successStyle
			}

			row = lipgloss.JoinHorizontal(lipgloss.Top,
				cellStyle.Width(30).Render(truncateString(result.URL, 27)),
				cellStyle.Width(10).Render(fmt.Sprintf("%d", result.PacketsSent)),
				recvStyle.Width(10).Render(fmt.Sprintf("%d", result.PacketsRecv)),
				lossStyle.Width(10).Render(fmt.Sprintf("%.1f%%", result.PacketLoss)),
				cellStyle.Width(18).Render(formatDuration(result.AvgRtt)),
			)
		}
		pingRows = append(pingRows, row)
	}

	// Render ping table
	pingTable := lipgloss.JoinVertical(lipgloss.Left, pingRows...)
	fmt.Println(tableStyle.Render(pingTable))

	// Print fetch results table
	fetchTitle := titleStyle.Render(" HTTP Fetch Results ")
	fmt.Println(lipgloss.NewStyle().Width(80).Align(lipgloss.Center).Render(fetchTitle))

	// Create fetch table header
	fetchTableHeader := []string{
		headerStyle.Width(30).Render("URL"),
		headerStyle.Width(12).Render("Status"),
		headerStyle.Width(12).Render("Size (MB)"),
		headerStyle.Width(24).Render("Notes"),
	}

	fetchHeaderRow := lipgloss.JoinHorizontal(lipgloss.Top, fetchTableHeader...)

	// Create fetch table rows
	var fetchRows []string
	fetchRows = append(fetchRows, fetchHeaderRow)

	for _, result := range allFetchResults {
		var statusStyle lipgloss.Style

		if result.Error != nil {
			row := lipgloss.JoinHorizontal(lipgloss.Top,
				cellStyle.Width(30).Render(truncateString(result.URL, 27)),
				errorStyle.Width(48).Render(fmt.Sprintf("Error: %v", result.Error)),
			)
			fetchRows = append(fetchRows, row)
			continue
		}

		// Style based on status code
		statusText := fmt.Sprintf("%d", result.StatusCode)
		if result.StatusCode >= 200 && result.StatusCode < 300 {
			statusStyle = successStyle
		} else if result.StatusCode >= 300 && result.StatusCode < 400 {
			statusStyle = warningStyle
			statusText += " (Redirect)"
		} else {
			statusStyle = errorStyle
		}

		notes := ""
		if len(result.Redirects) > 0 {
			notes = fmt.Sprintf("%d redirects", len(result.Redirects))
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			cellStyle.Width(30).Render(truncateString(result.URL, 27)),
			statusStyle.Width(12).Render(statusText),
			cellStyle.Width(12).Render(fmt.Sprintf("%.2f", result.BodySize)),
			cellStyle.Width(24).Render(notes),
		)
		fetchRows = append(fetchRows, row)
	}

	// Render fetch table
	fetchTable := lipgloss.JoinVertical(lipgloss.Left, fetchRows...)
	fmt.Println(tableStyle.Render(fetchTable))

	// Print detailed redirect information if any
	hasRedirects := false
	for _, result := range allFetchResults {
		if len(result.Redirects) > 0 {
			hasRedirects = true
			break
		}
	}

	if hasRedirects {
		redirectTitle := titleStyle.Render(" Redirect Details ")
		fmt.Println(lipgloss.NewStyle().Width(80).Align(lipgloss.Center).Render(redirectTitle))

		for _, result := range allFetchResults {
			if len(result.Redirects) > 0 {
				fmt.Println(infoStyle.Render(fmt.Sprintf(" → Redirects for %s:", result.URL)))
				for i, redirect := range result.Redirects {
					fmt.Println(cellStyle.Render(fmt.Sprintf("   %d. %s", i+1, redirect)))
				}
				fmt.Println()
			}
		}
	}
}

// Helper function to truncate long strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func pingUrl(url string, results chan<- PingResult) {
	result := PingResult{
		URL: url,
	}

	// Extract hostname from URL
	hostname := url
	if len(url) > 8 && url[:8] == "https://" {
		hostname = url[8:]
	} else if len(url) > 7 && url[:7] == "http://" {
		hostname = url[7:]
	}

	// Strip www. prefix if present
	if len(hostname) > 4 && hostname[:4] == "www." {
		hostname = hostname[4:]
	}

	result.Domain = hostname

	pinger, err := probing.NewPinger(hostname)
	if err != nil {
		result.Error = err
		results <- result
		return
	}

	// Set pinger options
	pinger.Count = 3
	pinger.Timeout = time.Second * 5
	// Need to set this for Windows
	pinger.SetPrivileged(true)

	err = pinger.Run()
	if err != nil {
		result.Error = err
		results <- result
		return
	}

	stats := pinger.Statistics()
	result.PacketsSent = stats.PacketsSent
	result.PacketsRecv = stats.PacketsRecv
	result.PacketLoss = stats.PacketLoss
	result.AvgRtt = stats.AvgRtt

	results <- result
}

func fetchData(url string, results chan<- FetchResult) {
	result := FetchResult{
		URL: url,
	}

	resp, err := http.Get(url)
	if err != nil {
		result.Error = err
		results <- result
		return
	}

	// Check if the response is a redirect
	for resp.StatusCode == 301 || resp.StatusCode == 302 {
		result.Redirects = append(result.Redirects, resp.Header.Get("Location"))
		resp, err = http.Get(resp.Header.Get("Location"))
		if err != nil {
			result.Error = err
			results <- result
			return
		}
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err
		results <- result
		return
	}

	bodySize := len(body)
	result.StatusCode = resp.StatusCode
	result.BodyLength = bodySize
	result.BodySize = float64(bodySize) / 1024 / 1024

	results <- result
}

// Helper function to format duration in a consistent way
func formatDuration(d time.Duration) string {
	// Convert everything to milliseconds for consistency
	ms := d.Milliseconds()
	return fmt.Sprintf("%.2f ms", float64(ms))
}
