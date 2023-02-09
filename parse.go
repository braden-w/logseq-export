package main

import (
	"fmt"
	"regexp"
)

func parseTextAndAttributes(rawContent string) (string, map[string]string) {
	// A set of regexes to parse the frontmatter, taking in the entire file content as a single string
	// // For Logseq Frontmatter
	// result := regexp.MustCompile(`^((?:.*?::.*\n)*)\n?((?:.|\s)+)$`).FindStringSubmatch(rawContent)
	// frontmatterArray := regexp.MustCompile(`(?m:^(.*?)::\s*(.*)$)`).FindAllStringSubmatch(result[1], -1)
	// For Obsidian Frontmatter
	result := regexp.MustCompile(`^---\n((?:.|\n)*?)\n---\n?((?:.|\s)+)$`).FindStringSubmatch(rawContent)

	// Print the result
	fmt.Println(rawContent)
	fmt.Println(result)

	frontmatterArray := regexp.MustCompile(`(?m:^(.*?):\s*(.*)$)`).FindAllStringSubmatch(result[1], -1)
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
