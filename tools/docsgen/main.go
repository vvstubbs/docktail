package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

type heading struct {
	Level int
	Text  string
	ID    string
}

type pageData struct {
	Title       string
	Description string
	Body        template.HTML
	Headings    []heading
	GeneratedAt string
}

func main() {
	sourceDir := flag.String("source", "docs", "directory containing canonical markdown files")
	websiteDir := flag.String("website", "website", "website output directory")
	flag.Parse()

	markdown, err := readMarkdown(*sourceDir)
	if err != nil {
		fatal(err)
	}

	rendered, headings, err := renderMarkdown([]byte(markdown))
	if err != nil {
		fatal(err)
	}

	title := "DockTail Documentation"
	if len(headings) > 0 && headings[0].Level == 1 {
		title = headings[0].Text
	}

	generatedAt := time.Now().UTC().Format("2006-01-02")
	data := pageData{
		Title:       title,
		Description: "Documentation for DockTail, a tool that automatically exposes Docker containers as Tailscale Services using label-based configuration.",
		Body:        template.HTML(rendered),
		Headings:    headings,
		GeneratedAt: generatedAt,
	}

	if err := os.MkdirAll(filepath.Join(*websiteDir, "docs"), 0o755); err != nil {
		fatal(err)
	}

	if err := writeFile(filepath.Join(*websiteDir, "docs", "index.html"), renderTemplate(data)); err != nil {
		fatal(err)
	}
	if err := writeFile(filepath.Join(*websiteDir, "docs.md"), generatedMarkdown(markdown)); err != nil {
		fatal(err)
	}
	if err := writeFile(filepath.Join(*websiteDir, "llms.txt"), renderLLMS(data)); err != nil {
		fatal(err)
	}
	if err := writeFile(filepath.Join(*websiteDir, "llms-full.txt"), renderLLMSFull(markdown)); err != nil {
		fatal(err)
	}
	if err := writeFile(filepath.Join(*websiteDir, "sitemap.xml"), renderSitemap(generatedAt)); err != nil {
		fatal(err)
	}
}

func readMarkdown(sourceDir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(sourceDir, "*.md"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no markdown files found in %s", sourceDir)
	}
	sort.Strings(matches)

	var out strings.Builder
	for i, path := range matches {
		body, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		if i > 0 {
			out.WriteString("\n\n")
		}
		out.Write(bytes.TrimSpace(body))
		out.WriteByte('\n')
	}
	return out.String(), nil
}

func renderMarkdown(source []byte) (string, []heading, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(goldhtml.WithUnsafe()),
	)

	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)
	headings := collectHeadings(doc, source)

	var rendered bytes.Buffer
	if err := md.Renderer().Render(&rendered, source, doc); err != nil {
		return "", nil, err
	}
	return rendered.String(), headings, nil
}

func collectHeadings(doc ast.Node, source []byte) []heading {
	var headings []heading
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := node.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		headings = append(headings, heading{
			Level: h.Level,
			Text:  strings.TrimSpace(headingText(h, source)),
			ID:    headingID(h),
		})
		return ast.WalkContinue, nil
	})
	return headings
}

func headingText(node ast.Node, source []byte) string {
	var out strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			out.Write(n.Segment.Value(source))
		case *ast.CodeSpan:
			for codeChild := n.FirstChild(); codeChild != nil; codeChild = codeChild.NextSibling() {
				if textNode, ok := codeChild.(*ast.Text); ok {
					out.Write(textNode.Segment.Value(source))
				}
			}
		default:
			out.WriteString(headingText(n, source))
		}
	}
	return out.String()
}

func headingID(node ast.Node) string {
	value, ok := node.AttributeString("id")
	if !ok {
		return ""
	}
	switch id := value.(type) {
	case []byte:
		return string(id)
	case string:
		return id
	default:
		return fmt.Sprint(id)
	}
}

func generatedMarkdown(markdown string) string {
	return "<!-- Generated from docs/*.md. Do not edit directly. -->\n\n" + markdown
}

func renderTemplate(data pageData) string {
	var out bytes.Buffer
	if err := docsTemplate.Execute(&out, data); err != nil {
		fatal(err)
	}
	return out.String()
}

func renderLLMS(data pageData) string {
	var out strings.Builder
	out.WriteString("# DockTail\n\n")
	out.WriteString("DockTail automatically exposes Docker containers as Tailscale Services using label-based configuration.\n\n")
	out.WriteString("## Primary URLs\n\n")
	out.WriteString("- Homepage: https://docktail.org/\n")
	out.WriteString("- Human documentation: https://docktail.org/docs/\n")
	out.WriteString("- Markdown documentation: https://docktail.org/docs.md\n")
	out.WriteString("- Full LLM documentation: https://docktail.org/llms-full.txt\n")
	out.WriteString("- Source code: https://github.com/marvinvr/docktail\n")
	out.WriteString("- Container image: ghcr.io/marvinvr/docktail:latest\n\n")
	out.WriteString("## Documentation Sections\n\n")
	for _, h := range data.Headings {
		if h.Level > 3 || h.ID == "" {
			continue
		}
		prefix := "-"
		if h.Level == 3 {
			prefix = "  -"
		}
		fmt.Fprintf(&out, "%s %s: https://docktail.org/docs/#%s\n", prefix, h.Text, h.ID)
	}
	out.WriteString("\n## Agent Guidance\n\n")
	out.WriteString("Read https://docktail.org/llms-full.txt when you need the complete documentation in one text document. Use https://docktail.org/docs.md when Markdown structure matters. Prefer the generated docs over README for complete configuration details.\n")
	return out.String()
}

func renderLLMSFull(markdown string) string {
	return "# DockTail Full Documentation\n\nSource: https://docktail.org/docs/\n\n" + markdown
}

func renderSitemap(lastmod string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://docktail.org/</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>https://docktail.org/docs/</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.9</priority>
  </url>
  <url>
    <loc>https://docktail.org/docs.md</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.7</priority>
  </url>
  <url>
    <loc>https://docktail.org/llms.txt</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.5</priority>
  </url>
  <url>
    <loc>https://docktail.org/llms-full.txt</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.5</priority>
  </url>
</urlset>
`, lastmod, lastmod, lastmod, lastmod, lastmod)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "docsgen: %v\n", err)
	os.Exit(1)
}

var docsTemplate = template.Must(template.New("docs").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Docs - DockTail</title>
  <meta name="description" content="{{ .Description }}">
  <meta property="og:title" content="Docs - DockTail">
  <meta property="og:description" content="{{ .Description }}">
  <meta property="og:type" content="website">
  <meta property="og:url" content="https://docktail.org/docs/">
  <meta property="og:image" content="https://docktail.org/assets/og-image.jpeg">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="Docs - DockTail">
  <meta name="twitter:description" content="{{ .Description }}">
  <meta name="twitter:image" content="https://docktail.org/assets/og-image.jpeg">
  <meta name="author" content="marvinvr">
  <meta name="keywords" content="docker, tailscale, containers, service mesh, tailnet, networking, devops, documentation">
  <link rel="canonical" href="https://docktail.org/docs/">
  <link rel="sitemap" type="application/xml" href="/sitemap.xml">
  <link rel="alternate" type="text/markdown" href="/docs.md" title="DockTail documentation in Markdown">
  <link rel="alternate" type="text/plain" href="/llms.txt" title="DockTail guide for agents">
  <link rel="alternate" type="text/plain" href="/llms-full.txt" title="DockTail full documentation for agents">
  <link rel="icon" href="data:image/svg+xml,&lt;svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'&gt;&lt;text y='.9em' font-size='90'&gt;🍸&lt;/text&gt;&lt;/svg&gt;">
  <script type="application/ld+json">
    {
      "@context": "https://schema.org",
      "@graph": [
        {
          "@type": "TechArticle",
          "headline": "{{ .Title }}",
          "description": "{{ .Description }}",
          "url": "https://docktail.org/docs/",
          "image": "https://docktail.org/assets/og-image.jpeg",
          "dateModified": "{{ .GeneratedAt }}",
          "author": {
            "@type": "Person",
            "name": "marvinvr",
            "url": "https://marvinvr.ch"
          },
          "about": ["Docker", "Tailscale", "Tailscale Services", "Container networking"],
          "isPartOf": {
            "@type": "WebSite",
            "name": "DockTail",
            "url": "https://docktail.org/"
          },
          "mainEntityOfPage": "https://docktail.org/docs/"
        },
        {
          "@type": "BreadcrumbList",
          "itemListElement": [
            {
              "@type": "ListItem",
              "position": 1,
              "name": "DockTail",
              "item": "https://docktail.org/"
            },
            {
              "@type": "ListItem",
              "position": 2,
              "name": "Docs",
              "item": "https://docktail.org/docs/"
            }
          ]
        }
      ]
    }
  </script>
  <script
    src="https://rybbit.marvinvr.ch/api/script.js"
    data-site-id="ef0b4c42a7c2"
    defer
  ></script>
  <style>
    :root {
      --bg: #f8fafc;
      --panel: #ffffff;
      --ink: #111827;
      --muted: #64748b;
      --line: #e5e7eb;
      --soft: #f1f5f9;
      --code: #0f172a;
      --link: #0f766e;
    }

    * { box-sizing: border-box; }
    html { scroll-behavior: smooth; scroll-padding-top: 5rem; }
    body {
      margin: 0;
      min-height: 100vh;
      background:
        radial-gradient(circle at top left, rgba(20, 184, 166, 0.10), transparent 28rem),
        linear-gradient(180deg, #ffffff 0, var(--bg) 22rem);
      color: var(--ink);
      font: 15px/1.65 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    }
    a { color: inherit; }
    .topbar {
      position: sticky;
      top: 0;
      z-index: 20;
      border-bottom: 1px solid var(--line);
      background: rgba(255, 255, 255, 0.92);
      backdrop-filter: blur(12px);
    }
    .topbar-inner {
      max-width: 1180px;
      height: 56px;
      margin: 0 auto;
      padding: 0 20px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 20px;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      text-decoration: none;
      font-weight: 800;
      letter-spacing: -0.04em;
    }
    .navlinks {
      display: flex;
      align-items: center;
      gap: 18px;
      color: var(--muted);
      font-size: 13px;
    }
    .navlinks a { text-decoration: none; }
    .navlinks a:hover { color: var(--ink); }
    .layout {
      max-width: 1180px;
      margin: 0 auto;
      padding: 34px 20px 56px;
      display: grid;
      grid-template-columns: 270px minmax(0, 1fr);
      gap: 42px;
    }
    .sidebar {
      position: sticky;
      top: 82px;
      align-self: start;
      max-height: calc(100vh - 100px);
      overflow: auto;
      padding-right: 8px;
    }
    .sidebar-title {
      margin: 0 0 12px;
      font-size: 12px;
      font-weight: 800;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }
    .toc {
      display: grid;
      gap: 2px;
      font-size: 13px;
    }
    .toc a {
      display: block;
      padding: 5px 10px;
      border-radius: 8px;
      color: var(--muted);
      text-decoration: none;
    }
    .toc a:hover,
    .toc a.active {
      background: var(--soft);
      color: var(--ink);
    }
    .toc .level-3 { padding-left: 24px; font-size: 12px; }
    .doc {
      min-width: 0;
      max-width: 830px;
      padding: 8px 0 0;
    }
    .doc h1 {
      margin: 0 0 14px;
      font-size: clamp(2.2rem, 8vw, 4.7rem);
      line-height: 0.95;
      letter-spacing: -0.08em;
    }
    .doc h2 {
      margin: 54px 0 14px;
      padding-top: 10px;
      border-top: 1px solid var(--line);
      font-size: 1.65rem;
      line-height: 1.15;
      letter-spacing: -0.05em;
    }
    .doc h3 {
      margin: 34px 0 10px;
      font-size: 1.1rem;
      letter-spacing: -0.03em;
    }
    .doc p,
    .doc ul,
    .doc ol,
    .doc table,
    .doc pre { margin: 0 0 18px; }
    .doc p,
    .doc li { color: #334155; }
    .doc strong { color: var(--ink); }
    .doc a {
      color: var(--link);
      text-decoration-thickness: 1px;
      text-underline-offset: 3px;
    }
    .doc ul,
    .doc ol { padding-left: 24px; }
    .doc li + li { margin-top: 6px; }
    .doc code {
      border: 1px solid var(--line);
      border-radius: 5px;
      background: var(--soft);
      color: #0f172a;
      padding: 1px 5px;
      font-size: 0.92em;
    }
    .doc pre {
      position: relative;
      overflow-x: auto;
      border-radius: 12px;
      background: var(--code);
      color: #e5e7eb;
      padding: 18px;
      box-shadow: 0 14px 36px rgba(15, 23, 42, 0.12);
    }
    .doc pre code {
      border: 0;
      border-radius: 0;
      background: transparent;
      color: inherit;
      padding: 0;
      font-size: 13px;
      line-height: 1.6;
    }
    .doc table {
      width: 100%;
      border-collapse: collapse;
      overflow: hidden;
      border: 1px solid var(--line);
      border-radius: 12px;
      background: var(--panel);
      display: block;
      overflow-x: auto;
    }
    .doc th,
    .doc td {
      padding: 10px 12px;
      border-bottom: 1px solid var(--line);
      text-align: left;
      vertical-align: top;
      white-space: nowrap;
    }
    .doc td:last-child { white-space: normal; min-width: 280px; }
    .doc th {
      background: var(--soft);
      font-size: 12px;
      color: var(--ink);
    }
    .doc tr:last-child td { border-bottom: 0; }
    .footer {
      max-width: 1180px;
      margin: 0 auto;
      padding: 26px 20px 40px;
      border-top: 1px solid var(--line);
      color: var(--muted);
      font-size: 12px;
      display: flex;
      justify-content: space-between;
      gap: 16px;
      flex-wrap: wrap;
    }
    .footer a { color: inherit; text-decoration: none; }
    .footer a:hover { color: var(--ink); }

    @media (max-width: 900px) {
      .layout { display: block; padding-top: 24px; }
      .sidebar {
        position: static;
        max-height: none;
        margin-bottom: 28px;
        padding: 14px;
        border: 1px solid var(--line);
        border-radius: 14px;
        background: rgba(255, 255, 255, 0.75);
      }
      .toc { grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); }
      .toc .level-3 { display: none; }
      .doc h2 { margin-top: 42px; }
    }
  </style>
</head>
<body>
  <nav class="topbar">
    <div class="topbar-inner">
      <a class="brand" href="/">docktail</a>
      <div class="navlinks">
        <a href="/docs.md">markdown</a>
        <a href="/llms.txt">agents</a>
        <a href="https://github.com/marvinvr/docktail">github</a>
      </div>
    </div>
  </nav>

  <div class="layout">
    <aside class="sidebar" aria-label="Documentation navigation">
      <p class="sidebar-title">On this page</p>
      <nav class="toc">
        {{ range .Headings }}
          {{ if and .ID (le .Level 3) (gt .Level 1) }}
            <a class="level-{{ .Level }}" href="#{{ .ID }}">{{ .Text }}</a>
          {{ end }}
        {{ end }}
      </nav>
    </aside>

    <main class="doc">
      {{ .Body }}
    </main>
  </div>

  <footer class="footer">
    <span>Generated from <code>docs/*.md</code> on {{ .GeneratedAt }}.</span>
    <span>
      <a href="/docs.md">Markdown</a>
      · <a href="/llms-full.txt">LLM full text</a>
      · <a href="https://github.com/marvinvr/docktail">GitHub</a>
    </span>
  </footer>

  <script>
    const links = [...document.querySelectorAll('.toc a')];
    const headings = links
      .map(link => document.getElementById(link.hash.slice(1)))
      .filter(Boolean);

    const observer = new IntersectionObserver(entries => {
      for (const entry of entries) {
        if (!entry.isIntersecting) continue;
        links.forEach(link => link.classList.toggle('active', link.hash === '#' + entry.target.id));
      }
    }, { rootMargin: '-20% 0px -70% 0px' });

    headings.forEach(heading => observer.observe(heading));
  </script>
</body>
</html>
`))
