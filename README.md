# fontawesome
https://fontawesome.com/icons

# Installation
`go get -u gitea.com/go-icon/fontawesome`

# Usage
```go
icon := fontawesome.Fa500pxBrands()

// Get the raw XML
xml := icon.XML()

// Get something suitable to pass directly to an html/template
html := icon.HTML()
```

# Build
`go generate generate.go`

# New Versions
To update the version of fontawesome, simply change `faVersion` in `fontawesome_generate.go` and re-build.

# License
[FA License](LICENSE.txt) (icons)
[MIT License](LICENSE) (library code)