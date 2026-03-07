// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import catppuccin from "@catppuccin/starlight";

const docsBase = (process.env.DOCS_BASE ?? '/jjui').replace(/\/$/, '');

// https://astro.build/config
export default defineConfig({
	site: 'https://idursun.github.io',
	base: docsBase,
	integrations: [
		starlight({
			title: 'jjui',
			sidebar: [
				{
					label: 'Jujutsu UI',
					items: [
						{ label: 'Overview', slug: 'overview' },
						{ label: 'Install', slug: 'install' },
						{ label: 'Migrating to v0.10', slug: 'migrating/v0_10' },
					],
				},
				{
					label: 'Customization',
					items: [
						{ label: 'config.toml', slug: 'customization/config-toml' },
						{ label: 'config.lua', slug: 'customization/config-lua' },
						{ label: 'Actions and Bindings', slug: 'customization/actions-and-bindings' },
						{ label: 'Lua Scripting', slug: 'customization/lua-scripting' },
						{ label: 'Themes', slug: 'customization/themes' },
						{ label: 'Command Line Options', slug: 'customization/command-line-options' },
					],
				},
				{
					label: 'Revisions',
					items: [
						{ label: 'Overview', slug: 'revisions' },
						{ label: 'Details', slug: 'details' },
						{ label: 'Rebase', slug: 'revisions/rebase' },
						{ label: 'Absorb', slug: 'revisions/absorb' },
						{ label: 'Abandon', slug: 'revisions/abandon' },
						{ label: 'Duplicate', slug: 'revisions/duplicate' },
						{ label: 'Squash', slug: 'revisions/squash' },
						{ label: 'Revert', slug: 'revisions/revert' },
						{ label: 'Set Parents', slug: 'revisions/set-parents' },
						{ label: 'Evolog', slug: 'revisions/evolog' },
						{ label: 'Inline Describe', slug: 'revisions/inline-describe' },
						{ label: 'Ace Jump', slug: 'ace-jump' },
					],
				},
				{
					label: 'Views',
					items: [
						{ label: 'Revset', slug: 'revset' },
						{ label: 'Oplog', slug: 'oplog' },
						{ label: 'Undo and Redo', slug: 'undo-redo' },
						{ label: 'Git', slug: 'git' },
						{ label: 'Bookmarks', slug: 'bookmarks' },
					],
				},
				{
					label: 'Tools',
					items: [
						{ label: 'Preview', slug: 'preview' },
						{ label: 'Diff Viewer', slug: 'diff-viewer' },
						{ label: 'Flash Messages', slug: 'flash-messages' },
						{ label: 'Command History', slug: 'command-history' },
						{ label: 'Fuzzy File Finder', slug: 'fuzzy-file-finder' },
						{ label: 'Command Execution', slug: 'command-execution' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Lua Cookbook', slug: 'lua-cookbook' },
					],
				},
			],
			components: {
				Sidebar: './src/components/Sidebar.astro',
				Header: './src/components/Header.astro',
				Footer: './src/components/Footer.astro',
				SocialIcons: './src/components/SocialIcons.astro',
				PageSidebar: './src/components/PageSidebar.astro',
			},
			plugins: [
				catppuccin({
					dark: { flavor: "macchiato", accent: "mauve" },
					light: { flavor: "latte", accent: "mauve" },
				}),
			],
			editLink: {
				baseUrl: 'https://github.com/idursun/jjui/edit/docs/',
			},
			customCss: [
				'./src/styles/custom.css',
			],
		}),
	],
});
