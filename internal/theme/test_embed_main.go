//go:build ignore

package main

import (
    "embed"
    "fmt"
    "path"
    
    "gopkg.in/yaml.v3"
)

//go:embed themes/*.yaml
var themesFS embed.FS

type ThemeConfig struct {
    Name string `yaml:"name"`
}

func main() {
    entries, err := themesFS.ReadDir("themes")
    if err != nil {
        fmt.Println("Error reading dir:", err)
        return
    }
    fmt.Println("Found entries:", len(entries))
    for _, e := range entries {
        // Use path.Join (forward slash) instead of filepath.Join
        data, err := themesFS.ReadFile(path.Join("themes", e.Name()))
        if err != nil {
            fmt.Printf("  - %s: ERROR reading: %v\n", e.Name(), err)
            continue
        }
        
        var theme ThemeConfig
        if err := yaml.Unmarshal(data, &theme); err != nil {
            fmt.Printf("  - %s: ERROR parsing: %v\n", e.Name(), err)
            continue
        }
        fmt.Printf("  - %s: OK (%s)\n", e.Name(), theme.Name)
    }
}
