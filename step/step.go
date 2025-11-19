package step

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-zglob"

	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-steplib/bitrise-step-generate-xcode-html-report/xctesthtmlreport"
)

const (
	htmlReportDirKey   = "BITRISE_HTML_REPORT_DIR"
	htmlReportInfoFile = "report-info.json"
)

type Input struct {
	TestDeployDir    string `env:"test_result_dir,required"`
	XcresultPatterns string `env:"xcresult_patterns"`
	Verbose          bool   `env:"verbose,opt[true,false]"`
}

type Config struct {
	TestDeployDir    string
	XcresultPatterns []string
}

type Result struct {
	HtmlReportDir string
}

type ReportInfo struct {
	Category string `json:"category"`
}

type ReportGenerator struct {
	envRepository env.Repository
	inputParser   stepconf.InputParser
	exporter      export.Exporter
	logger        log.Logger
	htmlGenerator xctesthtmlreport.Generator
}

func NewReportGenerator(
	envRepository env.Repository,
	inputParser stepconf.InputParser,
	exporter export.Exporter,
	logger log.Logger,
	generator xctesthtmlreport.Generator,
) ReportGenerator {
	return ReportGenerator{
		envRepository: envRepository,
		inputParser:   inputParser,
		exporter:      exporter,
		logger:        logger,
		htmlGenerator: generator,
	}
}

func (r *ReportGenerator) ProcessConfig() (*Config, error) {
	var input Input
	err := r.inputParser.Parse(&input)
	if err != nil {
		return &Config{}, err
	}

	stepconf.Print(input)
	r.logger.EnableDebugLog(input.Verbose)

	patterns := strings.Split(strings.TrimSpace(input.XcresultPatterns), "\n")
	var filteredPatterns []string

	for _, p := range patterns {
		pattern := strings.TrimSpace(p)
		if pattern == "" {
			continue
		}

		if !strings.HasSuffix(pattern, ".xcresult") {
			return nil, fmt.Errorf("pattern (%s) must filter for xcresult files", pattern)
		}

		filteredPatterns = append(filteredPatterns, pattern)
	}

	return &Config{
		TestDeployDir:    input.TestDeployDir,
		XcresultPatterns: filteredPatterns,
	}, nil
}

func (r *ReportGenerator) InstallDependencies() error {
	r.logger.Println()
	r.logger.Infof("Installing XCTestHTMLReport")
	err := r.htmlGenerator.Install()
	if err != nil {
		return fmt.Errorf("failed to install htmlGenerator tool: %w", err)
	}
	return nil
}

func (r *ReportGenerator) Run(config Config) (Result, error) {
	r.logger.Println()
	r.logger.Infof("Collecting xcresult files")

	patterns := []string{
		fmt.Sprintf("%s/**/*.xcresult", config.TestDeployDir),
	}
	if 0 < len(config.XcresultPatterns) {
		patterns = config.XcresultPatterns
	}

	paths, err := collectFilesWithPatterns(patterns)
	if err != nil {
		return Result{}, fmt.Errorf("failed to find all xcresult files: %w", err)
	}

	if len(paths) == 0 {
		r.logger.Printf("No files found.")

		return Result{
			HtmlReportDir: "",
		}, nil
	}

	r.logger.Printf("List of files:")
	for _, path := range paths {
		r.logger.Printf("- %s", path)
	}

	rootDir, err := r.htmlReportsRootDir()
	if err != nil {
		return Result{}, fmt.Errorf("failed to create test report directory: %w", err)
	}

	r.logger.Println()
	r.logger.Infof("Generating reports")

	for _, path := range paths {
		if err := r.generateTestReport(rootDir, path); err != nil {
			r.logger.Errorf("failed to generate test report (%s): %w", path, err)
		}
	}

	r.logger.Println()
	r.logger.Donef("Finished")

	return Result{
		HtmlReportDir: rootDir,
	}, nil
}

func (r *ReportGenerator) Export(result Result) error {
	return r.exporter.ExportOutput(htmlReportDirKey, result.HtmlReportDir)
}

func (r *ReportGenerator) generateTestReport(rootDir string, xcresultPath string) error {
	baseName := strings.TrimSuffix(filepath.Base(xcresultPath), filepath.Ext(xcresultPath))
	dirPath := filepath.Join(rootDir, baseName)
	err := os.Mkdir(dirPath, 0755)
	if err != nil {
		if os.IsExist(err) {
			r.logger.Warnf("Html report already exists for %s at %s", baseName, dirPath)
			return nil
		}
		return err
	}

	err = r.htmlGenerator.Generate(dirPath, xcresultPath)
	if err != nil {
		return fmt.Errorf("failed to generate html: %w", err)
	}

	if err := injectGoogleAnalytics(dirPath); err != nil {
		return fmt.Errorf("failed to inject Google Analytics: %w", err)
	}

	if err := moveAssets(xcresultPath, dirPath); err != nil {
		return fmt.Errorf("failed to move assets: %w", err)
	}

	if err := createReportInfo(dirPath); err != nil {
		return fmt.Errorf("failed to create report info file: %w", err)
	}

	return nil
}

func (r *ReportGenerator) htmlReportsRootDir() (string, error) {
	reportDir := r.envRepository.Get(htmlReportDirKey)
	if reportDir == "" {
		return os.MkdirTemp("", "html-reports")
	}

	exists, err := pathutil.NewPathChecker().IsDirExists(reportDir)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("html report dir (%s) does not exist or is not a folder", reportDir)
	}

	return reportDir, nil
}

func collectFilesWithPatterns(patterns []string) ([]string, error) {
	// Go does not have a set, so a map will help filter out duplicate results.
	allMatches := map[string]struct{}{}

	for _, pattern := range patterns {
		matches, err := zglob.Glob(pattern)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			allMatches[match] = struct{}{}
		}
	}

	var paths []string
	for key := range allMatches {
		paths = append(paths, key)
	}

	return paths, nil
}

func moveAssets(xcresultPath string, htmlReportDir string) error {
	entries, err := os.ReadDir(xcresultPath)
	if err != nil {
		return err
	}

	assetFolder := filepath.Join(htmlReportDir, filepath.Base(xcresultPath))
	if err := os.Mkdir(assetFolder, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		// The assets are dumped into the root, so we do not need the folders.
		if entry.IsDir() {
			continue
		}

		extension := filepath.Ext(entry.Name())
		// We want to only move the useful assets which are images and videos.
		if extension == ".plist" || extension == ".log" {
			continue
		}

		oldPath := filepath.Join(xcresultPath, entry.Name())
		newPath := filepath.Join(assetFolder, filepath.Base(entry.Name()))
		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}
	}

	return nil
}

func createReportInfo(htmlReportDir string) error {
	reportInfo := ReportInfo{
		Category: "test",
	}

	jsonData, err := json.Marshal(reportInfo)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(htmlReportDir, htmlReportInfoFile), jsonData, 0755); err != nil {
		return err
	}

	return nil
}

func injectGoogleAnalytics(htmlReportDir string) error {
	const googleAnalyticsScript = `<!-- Google tag (gtag.js) -->
<script async src="https://www.googletagmanager.com/gtag/js?id=G-VJV9NL05SD"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());

  gtag('config', 'G-VJV9NL05SD');
</script>
`

	// Find all HTML files in the directory
	htmlFiles, err := filepath.Glob(filepath.Join(htmlReportDir, "*.html"))
	if err != nil {
		return fmt.Errorf("failed to find HTML files: %w", err)
	}

	for _, htmlFile := range htmlFiles {
		content, err := os.ReadFile(htmlFile)
		if err != nil {
			return fmt.Errorf("failed to read HTML file %s: %w", htmlFile, err)
		}

		htmlContent := string(content)

		// Inject Google Analytics before </head>
		if strings.Contains(htmlContent, "</head>") {
			htmlContent = strings.Replace(htmlContent, "</head>", googleAnalyticsScript+"</head>", 1)
		} else {
			// If no </head> tag found, try to inject after <head>
			if strings.Contains(htmlContent, "<head>") {
				htmlContent = strings.Replace(htmlContent, "<head>", "<head>\n"+googleAnalyticsScript, 1)
			}
		}

		if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
			return fmt.Errorf("failed to write modified HTML file %s: %w", htmlFile, err)
		}
	}

	return nil
}
