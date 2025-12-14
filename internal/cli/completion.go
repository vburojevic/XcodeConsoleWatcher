package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
)

// CompletionCmd generates shell completions
type CompletionCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell type (bash, zsh, fish)"`
}

type completionNode struct {
	Subcommands []string
	Flags       []string
}

type completionIndex struct {
	Nodes      map[string]completionNode
	EnumByFlag map[string][]string // flag token (-f/--format/--foo) -> enum values
	KnownPaths []string
}

// Run executes the completion command.
//
// Note: we accept *kong.Context so completion output stays in sync with the actual CLI model.
func (c *CompletionCmd) Run(globals *Globals, ctx *kong.Context) error {
	model := (*kong.Node)(nil)
	if ctx != nil && ctx.Kong != nil && ctx.Model != nil {
		model = ctx.Model.Node
	}
	idx := buildCompletionIndex(model)

	switch c.Shell {
	case "bash":
		return c.generateBash(globals, idx)
	case "zsh":
		return c.generateZsh(globals, idx)
	case "fish":
		return c.generateFish(globals, idx)
	default:
		return fmt.Errorf("unsupported shell: %s", c.Shell)
	}
}

func buildCompletionIndex(model *kong.Node) completionIndex {
	// Be resilient when ctx/model isn't available (eg. tests or direct invocation).
	if model == nil {
		return completionIndex{
			Nodes:      map[string]completionNode{},
			EnumByFlag: map[string][]string{},
			KnownPaths: []string{""},
		}
	}

	nodes := map[string]completionNode{}
	enumByFlag := map[string][]string{}
	known := map[string]struct{}{"": {}}

	addEnums := func(tokens []string, rawEnum string) {
		rawEnum = strings.TrimSpace(rawEnum)
		if rawEnum == "" {
			return
		}
		values := strings.Split(rawEnum, ",")
		out := make([]string, 0, len(values))
		for _, v := range values {
			v = strings.TrimSpace(v)
			if v != "" {
				out = append(out, v)
			}
		}
		if len(out) == 0 {
			return
		}
		for _, token := range tokens {
			if token == "" {
				continue
			}
			// First writer wins (global flags show up everywhere).
			if _, ok := enumByFlag[token]; ok {
				continue
			}
			enumByFlag[token] = out
		}
	}

	var walk func(n *kong.Node, path []string)
	walk = func(n *kong.Node, path []string) {
		key := strings.Join(path, "__")
		known[key] = struct{}{}

		sub := make([]string, 0, len(n.Children))
		for _, child := range n.Children {
			if child == nil || child.Type != kong.CommandNode || child.Hidden {
				continue
			}
			sub = append(sub, child.Name)
			for _, a := range child.Aliases {
				if strings.TrimSpace(a) != "" {
					sub = append(sub, a)
				}
			}
		}
		sub = uniqueSorted(sub)

		// Flags available at this node (includes inherited/global flags).
		flagTokens := map[string]struct{}{}
		for _, group := range n.AllFlags(true) {
			for _, f := range group {
				if f == nil {
					continue
				}
				tokens := flagCompletionTokens(f)
				for _, t := range tokens {
					flagTokens[t] = struct{}{}
				}
				addEnums(tokens, f.Enum)
			}
		}
		flags := make([]string, 0, len(flagTokens))
		for t := range flagTokens {
			flags = append(flags, t)
		}
		sort.Strings(flags)

		nodes[key] = completionNode{
			Subcommands: sub,
			Flags:       flags,
		}

		for _, child := range n.Children {
			if child == nil || child.Type != kong.CommandNode || child.Hidden {
				continue
			}
			walk(child, append(path, child.Name))
		}
	}

	walk(model, nil)

	knownPaths := make([]string, 0, len(known))
	for k := range known {
		knownPaths = append(knownPaths, k)
	}
	sort.Strings(knownPaths)

	return completionIndex{
		Nodes:      nodes,
		EnumByFlag: enumByFlag,
		KnownPaths: knownPaths,
	}
}

func flagCompletionTokens(f *kong.Flag) []string {
	if f == nil {
		return nil
	}
	tokens := []string{"--" + f.Name}
	if f.Short != 0 {
		tokens = append(tokens, "-"+string(f.Short))
	}
	for _, a := range f.Aliases {
		a = strings.TrimSpace(a)
		if a != "" {
			tokens = append(tokens, "--"+a)
		}
	}
	return tokens
}

func uniqueSorted(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func quoteShellWords(words []string) string {
	// All command names and flags are safe (no spaces) but keep minimal quoting for correctness.
	out := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		out = append(out, w)
	}
	return strings.Join(out, " ")
}

func (c *CompletionCmd) generateBash(globals *Globals, idx completionIndex) error {
	var sb strings.Builder
	sb.WriteString(`# xcw bash completion script
# Add to ~/.bashrc or ~/.bash_profile:
#   eval "$(xcw completion bash)"

_xcw_complete_simulators() {
    local sims
    sims=$(xcrun simctl list devices booted -j 2>/dev/null | grep '"name"' | cut -d'"' -f4 | tr '\n' ' ')
    COMPREPLY=($(compgen -W "booted ${sims}" -- "${cur}"))
}

_xcw_is_cmdpath() {
    case "$1" in
`)
	for _, k := range idx.KnownPaths {
		if k == "" {
			continue
		}
		sb.WriteString("        ")
		sb.WriteString(k)
		sb.WriteString(")\n            return 0\n            ;;\n")
	}
	sb.WriteString(`        "")
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

_xcw_completions() {
    local cur prev words cword
    _init_completion || return

    local cmdpath=""
    local candidate=""
    local i
    for ((i=1; i < cword; i++)); do
        local w=${words[i]}
        [[ -z "${w}" ]] && continue
        [[ "${w}" == -* ]] && continue
        if [[ -z "${candidate}" ]]; then
            candidate="${w}"
        else
            candidate="${candidate}__${w}"
        fi
        if _xcw_is_cmdpath "${candidate}"; then
            cmdpath="${candidate}"
        else
            break
        fi
    done

    case "${prev}" in
        -s|--simulator)
            _xcw_complete_simulators
            return
            ;;
`)
	// Enum completion cases.
	enumTokens := make([]string, 0, len(idx.EnumByFlag))
	for token := range idx.EnumByFlag {
		enumTokens = append(enumTokens, token)
	}
	sort.Strings(enumTokens)

	for _, token := range enumTokens {
		values := idx.EnumByFlag[token]
		if len(values) == 0 {
			continue
		}
		sb.WriteString("        ")
		sb.WriteString(token)
		sb.WriteString(")\n            COMPREPLY=($(compgen -W \"")
		sb.WriteString(quoteShellWords(values))
		sb.WriteString("\" -- \"${cur}\"))\n            return\n            ;;\n")
	}

	sb.WriteString(`    esac

    local subcommands=""
    local flags=""
    case "${cmdpath}" in
`)
	// Per-path completion lists.
	paths := make([]string, 0, len(idx.Nodes))
	for k := range idx.Nodes {
		paths = append(paths, k)
	}
	sort.Strings(paths)

	for _, k := range paths {
		node := idx.Nodes[k]
		sb.WriteString("        \"")
		sb.WriteString(k)
		sb.WriteString("\")\n")
		sb.WriteString("            subcommands=\"")
		sb.WriteString(quoteShellWords(node.Subcommands))
		sb.WriteString("\"\n")
		sb.WriteString("            flags=\"")
		sb.WriteString(quoteShellWords(node.Flags))
		sb.WriteString("\"\n")
		sb.WriteString("            ;;\n")
	}

	sb.WriteString(`        *)
            subcommands=""
            flags=""
            ;;
    esac

    if [[ "${cur}" == -* ]]; then
        COMPREPLY=($(compgen -W "${flags}" -- "${cur}"))
        return
    fi

    if [[ -n "${subcommands}" ]]; then
        COMPREPLY=($(compgen -W "${subcommands}" -- "${cur}"))
        return
    fi
}

complete -F _xcw_completions xcw
`)

	_, err := fmt.Fprint(globals.Stdout, sb.String())
	return err
}

func (c *CompletionCmd) generateZsh(globals *Globals, idx completionIndex) error {
	// Keep this intentionally lightweight (no deep zsh _arguments trees).
	// This is generated from the Kong model to avoid command/flag drift.
	var sb strings.Builder
	sb.WriteString(`#compdef xcw
# xcw zsh completion script
# Add to ~/.zshrc:
#   eval "$(xcw completion zsh)"

_xcw_complete_simulators() {
  local -a sims
  sims=(booted ${(f)"$(xcrun simctl list devices booted -j 2>/dev/null | grep '\"name\"' | cut -d'\"' -f4)"})
  _describe 'simulator' sims
}

_xcw_is_cmdpath() {
  case "$1" in
`)
	for _, k := range idx.KnownPaths {
		if k == "" {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(k)
		sb.WriteString(") return 0;;\n")
	}
	sb.WriteString(`    "") return 0;;
    *) return 1;;
  esac
}

_xcw() {
  local cur prev cmdpath candidate
  cur="${words[CURRENT]}"
  prev="${words[CURRENT-1]}"

  cmdpath=""
  candidate=""
  local i
  for ((i=2; i < CURRENT; i++)); do
    local w="${words[i]}"
    [[ -z "${w}" ]] && continue
    [[ "${w}" == -* ]] && continue
    if [[ -z "${candidate}" ]]; then
      candidate="${w}"
    else
      candidate="${candidate}__${w}"
    fi
    if _xcw_is_cmdpath "${candidate}"; then
      cmdpath="${candidate}"
    else
      break
    fi
  done

  case "${prev}" in
    -s|--simulator)
      _xcw_complete_simulators
      return
      ;;
`)
	enumTokens := make([]string, 0, len(idx.EnumByFlag))
	for token := range idx.EnumByFlag {
		enumTokens = append(enumTokens, token)
	}
	sort.Strings(enumTokens)
	for _, token := range enumTokens {
		values := idx.EnumByFlag[token]
		if len(values) == 0 {
			continue
		}
		sb.WriteString("    ")
		sb.WriteString(token)
		sb.WriteString(")\n      _values '")
		sb.WriteString(token)
		sb.WriteString("' ")
		sb.WriteString(quoteShellWords(values))
		sb.WriteString("\n      return\n      ;;\n")
	}
	sb.WriteString(`  esac

  local -a subcommands
  local -a flags
  case "${cmdpath}" in
`)
	paths := make([]string, 0, len(idx.Nodes))
	for k := range idx.Nodes {
		paths = append(paths, k)
	}
	sort.Strings(paths)

	for _, k := range paths {
		node := idx.Nodes[k]
		sb.WriteString("    \"")
		sb.WriteString(k)
		sb.WriteString("\")\n")
		sb.WriteString("      subcommands=(")
		sb.WriteString(quoteShellWords(node.Subcommands))
		sb.WriteString(")\n")
		sb.WriteString("      flags=(")
		sb.WriteString(quoteShellWords(node.Flags))
		sb.WriteString(")\n")
		sb.WriteString("      ;;\n")
	}
	sb.WriteString(`    *)
      subcommands=()
      flags=()
      ;;
  esac

  if [[ "${cur}" == -* ]]; then
    compadd -- ${flags[@]}
    return
  fi

  if (( ${#subcommands[@]} > 0 )); then
    compadd -- ${subcommands[@]}
    return
  fi
}

compdef _xcw xcw
`)

	_, err := fmt.Fprint(globals.Stdout, sb.String())
	return err
}

func (c *CompletionCmd) generateFish(globals *Globals, idx completionIndex) error {
	var sb strings.Builder
	sb.WriteString(`# xcw fish completion script
# Add to ~/.config/fish/completions/xcw.fish

# Disable file completion by default
complete -c xcw -f

`)

	// Top-level commands (only) keep this fast and broadly compatible.
	root := idx.Nodes[""]
	for _, cmd := range root.Subcommands {
		sb.WriteString("complete -c xcw -n \"__fish_use_subcommand\" -a \"")
		sb.WriteString(cmd)
		sb.WriteString("\"\n")
	}

	// Global flags (those available at root).
	for _, flag := range root.Flags {
		if !strings.HasPrefix(flag, "--") {
			continue
		}
		long := strings.TrimPrefix(flag, "--")
		enum, hasEnum := idx.EnumByFlag[flag]
		if hasEnum && len(enum) > 0 {
			sb.WriteString("complete -c xcw -l ")
			sb.WriteString(long)
			sb.WriteString(" -xa \"")
			sb.WriteString(quoteShellWords(enum))
			sb.WriteString("\"\n")
			continue
		}
		sb.WriteString("complete -c xcw -l ")
		sb.WriteString(long)
		sb.WriteString("\n")
	}

	sb.WriteString(`
# Simulator completion (booted)
complete -c xcw -n "__fish_contains_opt -s s simulator" -a "(xcrun simctl list devices booted -j 2>/dev/null | grep '\"name\"' | cut -d'\"' -f4; echo booted)"
`)

	_, err := fmt.Fprint(globals.Stdout, sb.String())
	return err
}
