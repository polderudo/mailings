const path = require('path');
const fs = require('fs');

// Function to extract module path from go.mod
function getGoModulePath(moduleName) {
  try {
    const goModPath = path.join(__dirname, 'go.mod');
    const content = fs.readFileSync(goModPath, 'utf8');
    const match = content.match(new RegExp(`^replace\\s+${moduleName}\\s+=>\\s+(.+)$`, 'm'));
    if (match && match[1]) {
      return path.resolve(__dirname, match[1]);
    }
  } catch (e) {
    // go.mod file or module not found
  }

  // Fallback: search in go workspace or standard locations
  const homeDir = process.env.HOME || process.env.USERPROFILE;
  const possiblePaths = [
    path.join(homeDir, 'go/pkg/mod', moduleName.replace('/', '@')),
    path.join(homeDir, 'go', moduleName),
  ];

  for (const p of possiblePaths) {
    if (fs.existsSync(p)) {
      return p;
    }
  }

  return null;
}

const TEMPLUI_PATH = getGoModulePath('github.com/templui/templui');

const content = [
  "./views/**/*.templ",
  "./views/**/*.templ.go",
  "./ui-components/**/*.templ",
  "./ui-components/**/*.templ.go",
  "./ui-components/**/*.go",
  "./ui-components/**/*.js",
  "./templates/**/*.templ",
  "./templates/**/*.templ.go",
  "./templates/**/*.html",
  "./assets/js/**/*.js",
];

if (TEMPLUI_PATH && fs.existsSync(TEMPLUI_PATH)) {
  content.push(path.join(TEMPLUI_PATH, "components/**/*.templ"));
  content.push(path.join(TEMPLUI_PATH, "components/**/*.go"));
  content.push(path.join(TEMPLUI_PATH, "components/**/*.js"));
}

/** @type {import('tailwindcss').Config} */
module.exports = {
  content,
  safelist: [
    'bg-primary', 'text-primary', 'text-primary-foreground', 'border-primary', 'ring-primary',
    'bg-secondary', 'text-secondary', 'text-secondary-foreground', 'border-secondary', 'ring-secondary',
    'bg-accent', 'text-accent', 'text-accent-foreground', 'border-accent',
    'bg-muted', 'text-muted', 'text-muted-foreground', 'border-muted',
    'bg-destructive', 'text-destructive', 'border-destructive',
    'bg-info', 'text-info', 'border-info',
    'bg-success', 'text-success', 'border-success',
    'bg-warning', 'text-warning', 'border-warning',
    'bg-error', 'text-error', 'border-error',
    'bg-primary/5', 'bg-primary/10', 'bg-primary/15', 'bg-primary/20',
    'bg-secondary/5', 'bg-secondary/10', 'bg-secondary/15', 'bg-secondary/20', 'bg-secondary/80', 'bg-secondary/90',
    'border-primary/20', 'border-primary/50',
    'bg-destructive/90',
    'bg-accent/90', 'bg-accent/15', 'bg-accent/10',
    'hover:bg-primary', 'hover:bg-primary/90', 'hover:bg-primary/15',
    'hover:bg-secondary', 'hover:bg-secondary/80',
    'hover:bg-destructive', 'hover:bg-destructive/90',
    'hover:bg-accent', 'hover:bg-accent/90',
    'hover:opacity-90',
    'shadow-xs',
  ],
};
