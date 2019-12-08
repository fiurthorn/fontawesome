// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var (
	oFlag string

	faVersion = "5.11.2"
)

// These constants are used during generation of SetX functions.
const (
	widthAttrIndex = iota + 2
	heightAttrIndex
	styleAttrIndex
	idAttrIndex
	classAttrIndex
)

func init() {
	flag.StringVar(&oFlag, "o", "", "write output to `file` (default standard output)")
}

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	// Get icons
	resp, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/FortAwesome/Font-Awesome/%s/metadata/icons.json", faVersion))
	if err != nil {
		return fmt.Errorf("get icons: %v", err)
	}
	defer resp.Body.Close()

	// Generate

	var icons map[string]icon
	err = json.NewDecoder(resp.Body).Decode(&icons)
	if err != nil {
		return fmt.Errorf("decode icons: %v", err)
	}

	var names []string
	for name := range icons {
		names = append(names, name)
	}
	sort.Strings(names)

	var buf bytes.Buffer
	fmt.Fprint(&buf, `package fontawesome

import (
	"fmt"
	"html/template"
)

// Icons is a list of all FontAwesome icons
var Icons = []string{"`)

	fmt.Fprintf(&buf, strings.Join(names, `", "`))

	fmt.Fprintf(&buf, `"}

// FontAwesome represents an SVG node
type FontAwesome struct {
	xml    string
	width  int
	height int
	style  string
	id     string
	class  string
}

// XML returns the SVG node as an XML string
func (fa *FontAwesome) XML() string {
	return fmt.Sprintf(fa.xml, fa.width, fa.height, fa.style, fa.id, fa.class)
}

// HTML returns the SVG node as an HTML template, safe for use in Go templates
func (fa *FontAwesome) HTML() template.HTML {
	return template.HTML(fa.XML())
}

// Size sets the size of a FontAwesome icon
// Short for calling Width and Height with the same int
func (fa *FontAwesome) Size(size int) {
	fa.Width(size)
	fa.Height(size)
}

// Width sets the width of a FontAwesome icon
func (fa *FontAwesome) Width(width int) {
	fa.width = width
}

// Height sets the height of a FontAwesome icon
func (fa *FontAwesome) Height(height int) {
	fa.height = height
}

// Style sets the style of a FontAwesome icon
func (fa *FontAwesome) Style(style string) {
	fa.style = style
}

// Id sets the id of a FontAwesome icon
func (fa *FontAwesome) Id(id string) {
	fa.id = id
}

// Class sets the class of a FontAwesome icon
func (fa *FontAwesome) Class(class string) {
	fa.class = class
}

// Icon returns the named FontAwesome SVG node.
// It returns nil if name is not a valid FontAwesome symbol name.
func Icon(name string) *FontAwesome {
	switch name {
`)
	for _, name := range names {
		ico := icons[name]
		for _, style := range ico.Styles {
			fmt.Fprintf(&buf, "	case %q:\n		return %v()\n", name+"-"+style, kebab(name)+strings.Title(style))
		}
	}
	fmt.Fprint(&buf, `	default:
		return nil
	}
}
`)

	// Write all individual FontAwesome icon functions.
	for _, name := range names {
		generateAndWriteFontAwesome(&buf, icons, name)
	}

	var w io.Writer
	switch oFlag {
	case "":
		w = os.Stdout
	default:
		f, err := os.Create(oFlag)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	_, err = w.Write(buf.Bytes())
	return err
}

var faTemplate = template.Must(template.New("fontawesome").Parse(`return &FontAwesome{
	xml: ` + "`" + `{{.xml}}` + "`" + `,
	width: 16,
	height: 16,
	style: "display: inline-block; vertical-align: text-top; fill: currentColor;",
}
`))

type icon struct {
	Styles []string         `json:"styles"`
	SVG    map[string]style `json:"svg"`
}

type style struct {
	Raw string `json:"raw"`
}

func generateAndWriteFontAwesome(w io.Writer, icons map[string]icon, name string) {
	ico := icons[name]
	for _, style := range ico.Styles {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "// %s returns the %q FontAwesome icon.\n", kebab(name)+strings.Title(style), name+"-"+style)
		fmt.Fprintf(w, "func %s() *FontAwesome {\n", kebab(name)+strings.Title(style))
		st := ico.SVG[style]
		page, err := html.Parse(bytes.NewBufferString(st.Raw))
		if err != nil {
			fmt.Println(err)
			return
		}
		node := page.FirstChild.LastChild.FirstChild
		node.Attr = append(node.Attr,
			html.Attribute{Key: "width", Val: "%d"},
			html.Attribute{Key: "height", Val: "%d"},
			html.Attribute{Key: "style", Val: "%s"},
			html.Attribute{Key: "id", Val: "%s"},
			html.Attribute{Key: "class", Val: "%s"},
		)
		node.InsertBefore(&html.Node{
			FirstChild: &html.Node{
				Type:      html.TextNode,
				Data:      name,
				Namespace: "svg",
			},
			Type:      html.ElementNode,
			DataAtom:  atom.Title,
			Data:      "title",
			Namespace: "svg",
		}, node.FirstChild)

		xml := &strings.Builder{}
		if err := html.Render(xml, node); err != nil {
			fmt.Println(err)
			return
		}

		faTemplate.Execute(w, map[string]interface{}{
			"xml": xml.String(),
		})
		fmt.Fprintln(w, "}")
	}
}

func kebab(input string) string {
	parts := make([]string, strings.Count(input, "-")+1)
	if _, err := strconv.Atoi(string(input[0])); err == nil {
		input = fmt.Sprintf("fa%s", input)
	}
	for idx, part := range strings.Split(input, "-") {
		parts[idx] = strings.Title(part)
	}
	return strings.Join(parts, "")
}
