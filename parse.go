package main

import (
	"regexp"
)

func parseTextAndAttributes(rawContent string) (string, map[string]string) {
	// A set of regexes to parse the frontmatter, taking in the entire file content as a single string
	// // For Logseq Frontmatter
	// result := regexp.MustCompile(`^((?:.*?::.*\n)*)\n?((?:.|\s)+)$`).FindStringSubmatch(rawContent)
	// frontmatterArray := regexp.MustCompile(`(?m:^(.*?)::\s*(.*)$)`).FindAllStringSubmatch(result[1], -1)
	// For Obsidian Frontmatter
	result := regexp.MustCompile(`^---\n((?:.|\n)*?)\n---\n?((?:.|\s)+)$`).FindStringSubmatch(rawContent)

	// Captures with regexes are 1-indexed, so result[1] is the first capture group, result[2] is the second, etc.

	// Print the result
	// fmt.Println(rawContent)
	// fmt.Println(result)
	
	// If there is no frontmatter, return the entire content as the text and an empty map
	if len(result) == 0 {
		return rawContent, map[string]string{}
	}
	
	// result[1] is the inner frontmatter (the stuff between the ---)
	innerFrontmatter := result[1]

	// Remove all empty frontmatter from result[1]
	keyValues := regexp.MustCompile(`(?m:^(.*?):\s*$)`).ReplaceAllString(innerFrontmatter, "")

	frontmatterArray := regexp.MustCompile(`(?m:^(.*?):\s*(.*)$)`).FindAllStringSubmatch(keyValues, -1)
	attributes := map[string]string{}
	for _, attrStrings := range frontmatterArray {
		attributes[attrStrings[1]] = attrStrings[2]
	}
	return result[2], attributes
}

func parsePage(filename, rawContent string) page {
	text, attributes := parseTextAndAttributes(rawContent)
	return page{
		filename:   filename,
		attributes: attributes,
		text:       text,
	}
}
