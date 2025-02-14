package main

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func sanitizeFileName(orig string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9-_]+")
	return strings.ToLower(re.ReplaceAllString(strings.ReplaceAll(orig, " ", "-"), ""))
}

func generateFileName(originalName string, attributes map[string]string) string {
	if _, ok := attributes["slug"]; !ok {
		return sanitizeFileName(originalName)
	}

	return fmt.Sprintf("%s.md", attributes["slug"])
}

func addTitleIfMissing(p page) page {
	if p.attributes["title"] == "" {
		p.attributes["title"] = regexp.MustCompile(`\.[^.]*$`).ReplaceAllString(p.filename, "")
	}
	return p
}

func addFileName(p page) page {
	// filename := generateFileName(p.filename, p.attributes)
	filename := p.filename
	folder := filepath.Join(path.Split(p.attributes["folder"])) // the page property always uses `/` but the final delimiter is OS-dependent
	p.filename = filepath.Join(folder, filename)
	return p
}

func removeEmptyBulletPoints(from string) string {
	return regexp.MustCompile(`(?m:^\s*-\s*$)`).ReplaceAllString(from, "")
}

func firstBulletPointsToParagraphs(from string) string {
	return regexp.MustCompile(`(?m:^- )`).ReplaceAllString(from, "\n")
}

func secondToFirstBulletPoints(from string) string {
	return regexp.MustCompile(`(?m:^\t-)`).ReplaceAllString(from, "\n-")
}

func removeTabFromMultiLevelBulletPoints(from string) string {
	return regexp.MustCompile(`(?m:^\t{2,}-)`).ReplaceAllStringFunc(from, func(s string) string {
		return s[1:]
	})
}

func wikilinksToLinks(from string) string {
	return regexp.MustCompile(`\[\[(.+?)\]\]`).ReplaceAllStringFunc(from, func(s string) string {
		// If the wikilink has a pipe, use the text after the pipe as the link text
		// e.g. [[Some Page|Some Text]] -> [Some Text](some-page)
		if strings.Contains(s, "|") {
			match := regexp.MustCompile(`\[\[(.+?)\|(.+?)\]\]`).FindStringSubmatch(s)
			// Some Page
			page := match[1]
			text := match[2]
			return fmt.Sprintf("[%s](%s)", text, sanitizeFileName(page))
		}

		// Otherwise, use the page name as the link text
		match := regexp.MustCompile(`\[\[(.+?)\]\]`).FindStringSubmatch(s)
		page := match[1]
		return fmt.Sprintf("[%s](%s)", page, sanitizeFileName(page))
	})
}

const multilineBlocks = `\n?(- .*\n(?:  .*\n?)+)`

/*
Makes sure that code blocks and multiline blocks are without any extra characters at the start of the line

  - ```ts
    const hello = "world"
    ```

is changed to

```ts
const hello = "world"
```
*/
func unindentMultilineStrings(from string) string {
	return regexp.MustCompile(multilineBlocks).ReplaceAllStringFunc(from, func(s string) string {
		match := regexp.MustCompile(multilineBlocks).FindStringSubmatch(s)
		onlyBlock := match[1]
		replacement := regexp.MustCompile(`((?m:^[- ] ))`).ReplaceAllString(onlyBlock, "") // remove the leading spaces or dash
		replacedString := strings.Replace(s, onlyBlock, replacement, 1)
		return fmt.Sprintf("\n%s", replacedString) // add extra new line
	})
}

// onlyText turns text transformer into a page transformer
func onlyText(textTransformer func(string) string) func(page) page {
	return func(p page) page {
		p.text = textTransformer(p.text)
		return p
	}
}

func applyAll(from page, transformers ...func(page) page) page {
	result := from
	for _, t := range transformers {
		result = t(result)
	}
	return result
}

func blogAssetUrl(logseqURL, imagePrefixPath string) string {
	_, assetName := path.Split(logseqURL)
	return path.Join(imagePrefixPath, assetName)
}

/*
processMarkdownImages finds all markdown images with **relative** URL e.g. `![alt](../assets/image.png)`
it extracts the relative URL into a `page.assets“ array
it replaces the relative links with `imagePrefixPath“: `{imagePrefixPath}/image.png`
*/
func processMarkdownImages(imagePrefixPath string) func(page) page {
	return func(p page) page {
		assetRegexp := regexp.MustCompile(`!\[.*?]\((\.\.?/.+?)\)`)
		links := assetRegexp.FindAllStringSubmatch(p.text, -1)
		assets := make([]string, 0, len(links))
		for _, l := range links {
			assets = append(assets, l[1])
		}
		p.assets = assets
		textWithAssets := assetRegexp.ReplaceAllStringFunc(p.text, func(s string) string {
			match := assetRegexp.FindStringSubmatch(s)
			originalURL := match[1]
			blogURL := blogAssetUrl(originalURL, imagePrefixPath)
			return strings.Replace(s, originalURL, blogURL, 1)
		})
		p.text = textWithAssets

		// image from the attributes

		imageLink, ok := p.attributes["image"]
		if !ok {
			return p
		}

		if !regexp.MustCompile(`^\.\.?/`).MatchString(imageLink) {
			return p
		}

		p.assets = append(p.assets, imageLink)
		p.attributes["image"] = blogAssetUrl(imageLink, imagePrefixPath)
		return p
	}
}

func transformPage(p page, webAssetsPathPrefix string) page {
	if p.attributes == nil {
		p.attributes = map[string]string{}
	}
	return applyAll(
		p,
		addTitleIfMissing,
		addFileName,
		onlyText(removeEmptyBulletPoints),
		onlyText(unindentMultilineStrings),
		onlyText(firstBulletPointsToParagraphs),
		onlyText(secondToFirstBulletPoints),
		onlyText(removeTabFromMultiLevelBulletPoints),
		processMarkdownImages(webAssetsPathPrefix),
		onlyText(wikilinksToLinks),
	)
}
