---
title: "Creating Custom Templates"
description: "How to generate, customise, and host your own Wails v3 project templates"
---

Wails ships with a set of built-in templates, but you can create your own and share them with the community. A custom template is just a Git repository — once it is hosted publicly, anyone can scaffold a project from it with a single command.

## Generate a template skeleton

The `wails3 generate template` command produces a ready-to-customise template directory:

```
wails3 generate template -name MyTemplate
```

All flags:

| Flag          | Description                                           | Default           |
| ------------- | ----------------------------------------------------- | ----------------- |
| -name         | Template name (required)                              | —                 |
| -author       | Author name                                           | —                 |
| -description  | Short description shown in the CLI                    | —                 |
| -helpurl      | URL to documentation for this template                | —                 |
| -version      | Initial version                                       | v0.0.1            |
| -frontend     | Copy an existing frontend directory into the template | —                 |
| -dir          | Where to write the template directory                 | Current directory |

The generated directory looks like this:

```
MyTemplate/
├── template.yaml          # Template metadata — edit this
├── NEXTSTEPS.md           # Guidance for you as the template author — delete before publishing
├── README.md              # Shown to users after they create a project
├── main.go.tmpl           # Application entry point
├── greetservice.go        # Example Go service
├── go.mod.tmpl            # Go module file
├── go.sum.tmpl            # Go checksums
├── gitignore.tmpl         # Becomes .gitignore in generated projects
├── Taskfile.tmpl.yml      # Build task definitions
└── frontend/              # Your frontend code
```

## Configure template metadata

Open `template.yaml` to set your template's metadata:

```yaml
# yaml-language-server: $schema=https://v3.wails.io/schemas/template.v3.json
name: "My Template"
shortname: my-template
author: Your Name
description: A template with my preferred setup
helpurl: https://github.com/yourname/my-template
version: v1.0.0
wailsVersion: 3
```

The `wailsVersion` field is **required** and must be `3`.

## Template variables

These variables are available in any `.tmpl` file:

| Variable                | Description                              | Example                         |
| ----------------------- | ---------------------------------------- | ------------------------------- |
| {{.ProjectName}}        | Project name supplied by the user        | "MyApp"                         |
| {{.BinaryName}}         | Binary filename                          | "myapp"                         |
| {{.ProductName}}        | Product display name                     | "My Application"                |
| {{.ProductDescription}} | Product description                      | "An awesome application"        |
| {{.ProductVersion}}     | Product version                          | "1.0.0"                        |
| {{.ProductCompany}}     | Company / author name                    | "My Company Ltd"                |
| {{.ProductCopyright}}   | Copyright string                         | "Copyright 2024 My Company Ltd" |
| {{.ProductComments}}    | Additional product comments              | "Built with Wails"              |
| {{.ProductIdentifier}}  | Reverse-DNS product identifier           | "com.mycompany.myapp"           |
| {{.ModulePath}}         | Go module path                           | "github.com/you/myapp"          |
| {{.WailsVersion}}       | Wails version used to create the project | "3.0.0"                         |
| {{.Typescript}}         | true if the template name ends in -ts    | true                            |
| {{.Opn}}                | Literal {{ — escape inside templates     | {{                              |
| {{.Cls}}                | Literal }} — escape inside templates     | }}                              |

## Publish on GitHub

1. **Create a public GitHub repository** for your template. The repository root must contain `template.yaml`.
2. **Delete `NEXTSTEPS.md`** — this file is for template authors and must not appear in projects users create from your template.
3. **Commit and push** the template directory contents as the repository root.
4. **Tag a release** using semantic versioning.

Users can now create projects from your template:

```
# Latest commit on the default branch
wails3 init -n myapp -t https://github.com/yourname/my-template

# Pinned to a specific release tag
wails3 init -n myapp -t https://github.com/yourname/my-template@v1.0.0
```

> **Third-party template warning:** When a user installs a remote template, Wails displays a warning explaining that the template is third-party code.

## Best practices

* **Write a clear `README.md`** — this is shown to users after they create a project.
* **Fill in `helpurl`** — link to your repository or dedicated documentation.
* **Pin frontend dependency versions** in `package.json` to avoid broken installs.
* **Test before tagging** — create a fresh project from the tagged release before announcing it.
* **Keep `wailsVersion: 3`** — this field tells Wails which major version the template targets.
* **Update regularly** — keep dependencies current and test against new Wails releases.
