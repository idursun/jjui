//go:build e2e

package main

import (
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Lua_ChooseResumesWithSelectedValue(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.WriteConfigLua(`
function setup(config)
  config.action("choose-value", function()
    local selected = choose({
      options = { "item-a", "item-b" },
      title = "Lua choose",
    })
    if selected ~= nil then
      flash("lua-selected:" .. selected)
    end
  end, {
    desc = "Choose a value",
    key = "Y",
    scope = "revisions",
  })
end
`)

	h.Start("initial")
	h.Key("Y")
	h.WaitText("Lua choose")
	h.Key("/")
	h.Text("item-b")
	if _, err := h.session.WaitForStableScreen(h.ctx, 3, func(screen []string) bool {
		return screenContains(screen, "item-b") && !screenContains(screen, "item-a")
	}); err != nil {
		t.Fatalf("choose did not filter to item-b: %v", err)
	}
	h.Key("Enter")
	h.WaitNoText("Lua choose")
	h.WaitText("lua-selected:item-b")
	h.Quit()
}

func Test_Lua_SyncJJCreatesAndRefreshesRevision(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.WriteConfigLua(`
function setup(config)
  config.action("scripted-new", function()
    local current = context.change_id()
    local _, err = jj("new", "-r", current)
    if err ~= nil then
      flash({text = "lua-new-error:" .. err, error = true})
      return
    end
    local id
    id, err = jj("log", "-r", "@", "-T", "change_id", "--no-graph", "--color", "never", "--quiet")
    if err ~= nil then
      flash({text = "lua-new-error:" .. err, error = true})
      return
    end
    revisions.refresh({selected_revision = id})
    flash("lua-new-complete")
  end, {
    desc = "Create a revision with Lua",
    key = "Y",
    scope = "revisions",
  })
end
`)

	h.Start("initial")
	before := countRevisions(t, h.Repo(), h.Env())
	h.Key("Y")
	h.WaitText("lua-new-complete")
	if got := countRevisions(t, h.Repo(), h.Env()); got != before+1 {
		t.Fatalf("revision count after synchronous Lua jj command = %d, want %d", got, before+1)
	}
	expected := strings.TrimSpace(runCommand(t, h.Repo(), h.Env(), "jj", "log", "-r", "@", "-T", "change_id", "--no-graph", "--color", "never", "--quiet"))
	h.WaitText("@  " + expected[:8])
	h.Quit()
}

func Test_Lua_DescribeWithInput(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	const description = "entered through lua input"
	h.WriteConfigLua(`
function setup(config)
  config.action("input-describe", function()
    local description = input({
      title = "Lua description",
      prompt = "Description",
    })
    if description == nil then
      flash("lua-input-cancelled")
      return
    end
    local _, err = jj("describe", "-m", description)
    if err ~= nil then
      flash({text = "lua-input-error:" .. err, error = true})
      return
    end
    revisions.refresh()
    flash("lua-input-described:" .. description)
  end, {
    desc = "Describe from Lua input",
    key = "Y",
    scope = "revisions",
  })
end
`)

	h.Start("initial")
	h.Key("Y")
	h.WaitText("Lua description")
	h.Text(description)
	h.Key("Enter")
	h.WaitText("lua-input-described:" + description)
	if got := workingCopyDescription(t, h.Repo(), h.Env()); got != description {
		t.Fatalf("working-copy description = %q, want %q", got, description)
	}
	h.Quit()
}

func Test_Lua_UpdatesRevsetFromSelection(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	expected := strings.TrimSpace(runCommand(t, h.Repo(), h.Env(), "jj", "log", "-r", "@", "-T", "change_id.shortest()", "--no-graph", "--color", "never", "--quiet"))
	h.WriteConfigLua(`
function setup(config)
  config.action("append selected ancestors", function()
    local change_id = context.change_id()
    local updated = revset.current() .. " | ancestors(" .. change_id .. ", 1)"
    revset.set(updated)
    jjui.wait_refresh()
    flash("lua-revset-complete:" .. change_id)
  end, {
    desc = "Append selected ancestors",
    key = "Y",
    scope = "revisions",
  })
end
`)

	h.Start("initial")
	h.Key("Y")
	h.WaitText("lua-revset-complete:" + expected)
	h.WaitText("ancestors(" + expected + ", 1)")
	h.Quit()
}

func Test_Lua_InlineDescribeThenCreateRevision(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	const description = "created by lua workflow"
	h.WriteConfigLua(`
function setup(config)
  config.action("describe and create", function()
    flash("lua-inline-describe-start")
    revisions.open_inline_describe()
    if not wait_close() then
      flash("lua-loop-cancelled")
      return
    end
    wait_refresh()
    revisions.new()
    wait_refresh()
    revisions.jump_to_working_copy()
    flash("lua-loop-complete")
  end, {
    desc = "Describe and create a revision",
    key = "Y",
    scope = "revisions",
  })
end
`)

	h.Start("initial")
	before := countRevisions(t, h.Repo(), h.Env())
	h.Key("Y")
	h.WaitText("lua-inline-describe-start")
	h.Text(description)
	h.Key("Enter")
	h.WaitText(description)
	if err := h.session.SendKey(ghostty.KeyEnter, "", ghostty.ModAlt); err != nil {
		t.Fatalf("apply inline description: %v", err)
	}
	h.WaitText("lua-loop-complete")

	if got := workingCopyDescription(t, h.Repo(), h.Env()); got != "" {
		t.Fatalf("working-copy description after loop = %q, want empty new working copy", got)
	}
	if got := countRevisions(t, h.Repo(), h.Env()); got != before+1 {
		t.Fatalf("revision count after Lua loop = %d, want %d", got, before+1)
	}
	h.Quit()
}

func Test_Lua_LoadsGlobalAndRepositoryConfigs(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.WriteGlobalConfigLua(`
function setup(config)
  config.action("global-action", function()
    flash("lua-global-ok")
  end, { key = "G", scope = "revisions", desc = "Global action" })
end
`)
	h.WriteRepoConfigLua(`
function setup(config)
  config.action("repo-action", function()
    flash("lua-repo-ok")
  end, { key = "R", scope = "revisions", desc = "Repository action" })
end
`)
	h.UseRepositoryConfig()

	h.Start("initial")
	h.Key("G")
	h.WaitText("lua-global-ok")
	h.Key("R")
	h.WaitText("lua-repo-ok")
	h.Quit()
}

func Test_Lua_RuntimeErrorsReachRenderedScreen(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.WriteConfigLua(`
function setup(config)
  config.action("runtime-error", function()
    error("lua-runtime-marker")
  end, { key = "Y", scope = "revisions", desc = "Runtime error" })
end
`)

	h.Start("initial")
	h.Key("Y")
	h.WaitText("lua-runtime-marker")
	h.Quit()
}

func Test_Lua_StartupErrorsAreReported(t *testing.T) {
	t.Parallel()
	t.Run("syntax", func(t *testing.T) {
		h := NewHarness(t)
		h.WriteConfigLua("function setup(config) this is not valid end")
		h.StartProcess()
		h.ExpectStartupError("Error in config.lua")
	})
	t.Run("setup", func(t *testing.T) {
		h := NewHarness(t)
		h.WriteConfigLua(`
function setup(config)
  error("lua-setup-marker")
end
`)
		h.StartProcess()
		h.ExpectStartupError("Error in config.lua", "lua-setup-marker")
	})
}
