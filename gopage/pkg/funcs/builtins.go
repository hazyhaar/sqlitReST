package funcs

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"zombiezen.com/go/sqlite"
)

const gopageVersion = "0.1.0"

// funcVersion returns the GoPage version
func funcVersion(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	return gopageVersion, nil
}

// funcUUID generates a UUID v4
func funcUUID(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		return nil, err
	}

	// Set version (4) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}

// funcSHA256 returns SHA256 hash of input
func funcSHA256(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	input := args[0].Text()
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:]), nil
}

// funcMD5 returns MD5 hash of input
func funcMD5(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	input := args[0].Text()
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:]), nil
}

// funcBase64Encode encodes input to base64
func funcBase64Encode(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	input := args[0].Text()
	return base64.StdEncoding.EncodeToString([]byte(input)), nil
}

// funcBase64Decode decodes base64 input
func funcBase64Decode(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	input := args[0].Text()
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, err
	}
	return string(decoded), nil
}

// funcURLEncode URL encodes input
func funcURLEncode(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	input := args[0].Text()
	return url.QueryEscape(input), nil
}

// funcURLDecode URL decodes input
func funcURLDecode(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	input := args[0].Text()
	decoded, err := url.QueryUnescape(input)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

// funcJSONExtract extracts value from JSON using path
func funcJSONExtract(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	jsonStr := args[0].Text()
	path := args[1].Text()

	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	// Simple path extraction (supports $.key.subkey format)
	path = strings.TrimPrefix(path, "$.")
	parts := strings.Split(path, ".")

	current := data
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil, nil
		}
	}

	switch v := current.(type) {
	case string:
		return v, nil
	case float64:
		return v, nil
	case bool:
		return v, nil
	case nil:
		return nil, nil
	default:
		// Return as JSON string for complex types
		b, _ := json.Marshal(v)
		return string(b), nil
	}
}

// funcSlugify converts text to URL-friendly slug
func funcSlugify(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	input := args[0].Text()

	// Convert to lowercase
	slug := strings.ToLower(input)

	// Replace accented characters
	replacements := map[rune]string{
		'à': "a", 'â': "a", 'ä': "a", 'á': "a", 'ã': "a",
		'è': "e", 'ê': "e", 'ë': "e", 'é': "e",
		'ì': "i", 'î': "i", 'ï': "i", 'í': "i",
		'ò': "o", 'ô': "o", 'ö': "o", 'ó': "o", 'õ': "o",
		'ù': "u", 'û': "u", 'ü': "u", 'ú': "u",
		'ç': "c", 'ñ': "n",
	}

	var builder strings.Builder
	for _, r := range slug {
		if replacement, ok := replacements[r]; ok {
			builder.WriteString(replacement)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			builder.WriteRune('-')
		}
	}

	// Clean up multiple dashes
	slug = builder.String()
	re := regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	return slug, nil
}

// funcTruncate truncates text to specified length with ellipsis
func funcTruncate(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	input := args[0].Text()
	maxLen := int(args[1].Int64())

	if len(input) <= maxLen {
		return input, nil
	}

	// Find last space before maxLen to avoid cutting words
	truncated := input[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "...", nil
}

// funcStripHTML removes HTML tags from text
func funcStripHTML(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	input := args[0].Text()

	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	stripped := re.ReplaceAllString(input, "")

	// Decode HTML entities
	stripped = html.UnescapeString(stripped)

	// Clean up whitespace
	re = regexp.MustCompile(`\s+`)
	stripped = re.ReplaceAllString(stripped, " ")
	stripped = strings.TrimSpace(stripped)

	return stripped, nil
}

// funcMarkdownToHTML converts basic markdown to HTML
func funcMarkdownToHTML(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	input := args[0].Text()

	// Escape HTML first
	output := html.EscapeString(input)

	// Headers
	output = regexp.MustCompile(`(?m)^### (.+)$`).ReplaceAllString(output, "<h3>$1</h3>")
	output = regexp.MustCompile(`(?m)^## (.+)$`).ReplaceAllString(output, "<h2>$1</h2>")
	output = regexp.MustCompile(`(?m)^# (.+)$`).ReplaceAllString(output, "<h1>$1</h1>")

	// Bold and italic
	output = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(output, "<strong>$1</strong>")
	output = regexp.MustCompile(`\*(.+?)\*`).ReplaceAllString(output, "<em>$1</em>")
	output = regexp.MustCompile(`__(.+?)__`).ReplaceAllString(output, "<strong>$1</strong>")
	output = regexp.MustCompile(`_(.+?)_`).ReplaceAllString(output, "<em>$1</em>")

	// Code
	output = regexp.MustCompile("`(.+?)`").ReplaceAllString(output, "<code>$1</code>")

	// Links
	output = regexp.MustCompile(`\[(.+?)\]\((.+?)\)`).ReplaceAllString(output, `<a href="$2">$1</a>`)

	// Line breaks to paragraphs
	paragraphs := strings.Split(output, "\n\n")
	for i, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" && !strings.HasPrefix(p, "<h") {
			paragraphs[i] = "<p>" + p + "</p>"
		}
	}
	output = strings.Join(paragraphs, "\n")

	return output, nil
}

// funcNowUTC returns current UTC timestamp
func funcNowUTC(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	return time.Now().UTC().Format(time.RFC3339), nil
}

// funcFormatDate formats date according to layout
func funcFormatDate(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	dateStr := args[0].Text()
	layout := args[1].Text()

	// Parse common date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
	}

	var t time.Time
	var err error
	for _, fmt := range formats {
		t, err = time.Parse(fmt, dateStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	// Convert layout shortcuts
	switch layout {
	case "short":
		layout = "02/01/2006"
	case "long":
		layout = "2 January 2006"
	case "time":
		layout = "15:04"
	case "datetime":
		layout = "02/01/2006 15:04"
	case "iso":
		layout = time.RFC3339
	}

	return t.Format(layout), nil
}

// funcTimeAgo returns human-readable time ago string
func funcTimeAgo(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	dateStr := args[0].Text()

	// Parse the date
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	var t time.Time
	var err error
	for _, fmt := range formats {
		t, err = time.Parse(fmt, dateStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return dateStr, nil // Return original if can't parse
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "à l'instant", nil
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "il y a 1 minute", nil
		}
		return fmt.Sprintf("il y a %d minutes", mins), nil
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "il y a 1 heure", nil
		}
		return fmt.Sprintf("il y a %d heures", hours), nil
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "hier", nil
		}
		return fmt.Sprintf("il y a %d jours", days), nil
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "il y a 1 semaine", nil
		}
		return fmt.Sprintf("il y a %d semaines", weeks), nil
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "il y a 1 mois", nil
		}
		return fmt.Sprintf("il y a %d mois", months), nil
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "il y a 1 an", nil
		}
		return fmt.Sprintf("il y a %d ans", years), nil
	}
}

// funcCoalesceEmpty returns first non-empty string argument
func funcCoalesceEmpty(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	for _, arg := range args {
		val := arg.Text()
		if val != "" {
			return val, nil
		}
	}
	return nil, nil
}

// funcFormatNumber formats number with thousands separator
func funcFormatNumber(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	num := args[0].Int64()

	// Format with spaces as thousands separator (French style)
	str := fmt.Sprintf("%d", num)
	result := make([]byte, 0, len(str)+(len(str)-1)/3)

	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ' ')
		}
		result = append(result, byte(c))
	}

	return string(result), nil
}

// funcFormatBytes formats bytes as human readable
func funcFormatBytes(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	bytes := args[0].Int64()

	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f To", float64(bytes)/TB), nil
	case bytes >= GB:
		return fmt.Sprintf("%.2f Go", float64(bytes)/GB), nil
	case bytes >= MB:
		return fmt.Sprintf("%.2f Mo", float64(bytes)/MB), nil
	case bytes >= KB:
		return fmt.Sprintf("%.2f Ko", float64(bytes)/KB), nil
	default:
		return fmt.Sprintf("%d o", bytes), nil
	}
}
