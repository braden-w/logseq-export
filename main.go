package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/exp/slices"
)

type page struct {
	filename   string
	attributes map[string]string
	assets     []string
	text       string
}

type exportOptions struct {
	graphPath           string
	blogFolder          string
	assetsRelativePath  string
	webAssetsPathPrefix string
	unquotedProperties  []string
}

/*
findMatchingFiles finds all files in rootPath that contain substring
ignoreRegexp is an expression that is evaluated on **relative** path of files within the graph (e.g. `.git/HEAD` or `logseq/.bkp/something.md`) if it matches, the file is not processed
*/
func findMatchingFiles(appFS afero.Fs, rootPath string, substring string, ignoreRegexp *regexp.Regexp) ([]string, error) {
	var result []string
	err := afero.Walk(appFS, rootPath, func(path string, info fs.FileInfo, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if info.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
		if ignoreRegexp != nil && ignoreRegexp.MatchString(filepath.ToSlash(relativePath)) {
			return nil
		}
		file, err := appFS.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}
		defer file.Close()
		// If the file directory contains the substring and is a markdown file, add it to the result
		if strings.Contains(relativePath, substring) && strings.HasSuffix(path, ".md") {
			// Print "Hit"
			fmt.Println(relativePath)
			result = append(result, path)
			// Print the path and stop scanning the file
			// fmt.Println(path)
			return nil
		}

		// fileScanner := bufio.NewScanner(file)
		// for fileScanner.Scan() {
		// 	line := fileScanner.Text()
		// 	if strings.Contains(line, substring) {
		// 		result = append(result, path)
		// 		// Print the path and stop scanning the file
		// 		// fmt.Println(path)
		// 		return nil
		// 	}
		// }
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

/*
parseOptions produce a valid exportOptions object
if a mandatory argument is missing, parseOptions will print error message, program usage and exits with os.Exit(1)
*/
func parseOptions() exportOptions {
	graphPath := flag.String("graphPath", "", "[MANDATORY] Folder where all public pages are exported.") // TODO rename graphPath -> graphFolder or maybe logseqFolder
	blogFolder := flag.String("blogFolder", "", "[MANDATORY] Folder where this program creates a new subfolder with public logseq pages.")
	assetsRelativePath := flag.String("assetsRelativePath", "logseq-images", "relative path within blogFolder where the assets (images) should be stored (e.g. 'static/images/logseq'). Default is `logseq-images`")
	webAssetsPathPrefix := flag.String("webAssetsPathPrefix", "/logseq-images", "path that the images are going to be served on on the web (e.g. '/public/images/logseq'). Default is `/logseq-images`")
	rawUnquotedProperties := flag.String("unquotedProperties", "", "comma-separated list of logseq page properties that won't be quoted in the markdown front matter, e.g. 'date,public,slug")
	flag.Parse()
	if *graphPath == "" || *blogFolder == "" {
		log.Println("mandatory argument is missing")
		flag.Usage()
		os.Exit(1)
	}
	unquotedProperties := parseUnquotedProperties(*rawUnquotedProperties)
	return exportOptions{
		graphPath:           *graphPath,
		blogFolder:          *blogFolder,
		assetsRelativePath:  *assetsRelativePath,
		webAssetsPathPrefix: *webAssetsPathPrefix,
		unquotedProperties:  unquotedProperties,
	}
}

func main() {
	appFS := afero.NewOsFs()
	options := parseOptions()
	// third argument is a substring that is searched in the file, fourth argument is for ignored files
	publicFiles, err := findMatchingFiles(appFS, options.graphPath, "Content/", regexp.MustCompile(`^(.obsidian|logseq|.git|ignore-compile)/`))
	if err != nil {
		log.Fatalf("Error during walking through a folder %v", err)
	}
	for _, publicFile := range publicFiles {
		err = exportPublicPage(appFS, publicFile, options)
		if err != nil {
			log.Fatalf("Error when exporting page %q: %v", publicFile, err)
		}
	}
}

func exportPublicPage(appFS afero.Fs, publicFile string, options exportOptions) error {
	srcContent, err := afero.ReadFile(appFS, publicFile)
	if err != nil {
		return fmt.Errorf("reading the %q file failed: %v", publicFile, err)
	}
	_, name := filepath.Split(publicFile)
	page := parsePage(name, string(srcContent))
	result := transformPage(page, options.webAssetsPathPrefix)
	assetFolder := filepath.Join(options.blogFolder, options.assetsRelativePath)
	err = copyAssets(appFS, publicFile, assetFolder, result.assets)
	if err != nil {
		return fmt.Errorf("copying assets for page %q failed: %v", publicFile, err)
	}
	dest := filepath.Join(options.blogFolder, result.filename)
	folder, _ := filepath.Split(dest)
	err = appFS.MkdirAll(folder, os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating parent directory for %q failed: %v", dest, err)
	}
	outputFileContent := render(result, options.unquotedProperties)
	err = afero.WriteFile(appFS, dest, []byte(outputFileContent), 0644)
	if err != nil {
		return fmt.Errorf("copying file %q failed: %v", dest, err)
	}
	return nil
}

func copyAssets(appFS afero.Fs, baseFile string, assetFolder string, assets []string) error {
	err := appFS.MkdirAll(assetFolder, os.ModePerm)
	if err != nil {
		log.Fatalf("Error when making assets folder %q: %v", assetFolder, err)
	}
	baseDir, _ := filepath.Split(baseFile)
	for _, relativeAssetPath := range assets {
		assetPath := filepath.Clean(filepath.Join(baseDir, relativeAssetPath))
		_, assetName := filepath.Split(assetPath)
		destPath := filepath.Join(assetFolder, assetName)
		err := copyFile(appFS, assetPath, destPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseUnquotedProperties(param string) []string {
	if param == "" {
		return []string{}
	}
	return strings.Split(param, ",")
}

func render(p page, dontQuote []string) string {
	sortedKeys := make([]string, 0, len(p.attributes))
	for k := range p.attributes {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)
	attributeBuilder := strings.Builder{}
	for _, key := range sortedKeys {
		if slices.Contains(dontQuote, key) {
			attributeBuilder.WriteString(fmt.Sprintf("%s: %s\n", key, p.attributes[key]))
		} else {
			attributeBuilder.WriteString(fmt.Sprintf("%s: %q\n", key, p.attributes[key]))
		}
	}
	return fmt.Sprintf("---\n%s---\n%s", attributeBuilder.String(), p.text)
}
