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
						{ label: 'Rebase', slug: 'revisions/rebase' },
						{ label: 'Absorb', slug: 'revisions/absorb' },
						{ label: 'Abandon', slug: 'revisions/abandon' },
						{ label: 'Duplicate', slug: 'revisions/duplicate' },
						{ label: 'Squash', slug: 'revisions/squash' },
						{ label: 'Revert', slug: 'revisions/revert' },
						{ label: 'Set Parents', slug: 'revisions/set-parents' },
						{ label: 'Evolog', slug: 'revisions/evolog' },
						{ label: 'Inline Describe', slug: 'revisions/inline-describe' },
					],
				},
				{
					label: 'Git',
					items: [
						{ label: 'Git', slug: 'git' },
					],
				},
				{
					label: 'Bookmarks',
					items: [
						{ label: 'Bookmarks', slug: 'bookmarks' },
					],
				},
				{
					label: 'Undo and Redo',
					items: [
						{ label: 'Undo and Redo', slug: 'undo-redo' },
					],
				},
				{
					label: 'Revset',
					items: [
						{ label: 'Revset', slug: 'revset' },
					],
				},
				{
					label: 'Oplog',
					items: [
						{ label: 'Oplog', slug: 'oplog' },
					],
				},
				{
					label: 'Command Execution',
					items: [
						{ label: 'Command Execution', slug: 'command-execution' },
					],
				},
				{
					label: 'Command History',
					items: [
						{ label: 'Command History', slug: 'command-history' },
					],
				},
				{
					label: 'Preview',
					items: [
						{ label: 'Preview', slug: 'preview' },
					],
				},
				{
					label: 'Details',
					items: [
						{ label: 'Details', slug: 'details' },
					],
				},
				{
					label: 'Fuzzy File Finder',
					items: [
						{ label: 'Fuzzy File Finder', slug: 'fuzzy-file-finder' },
					],
				},
				{
					label: 'Ace Jump',
					items: [
						{ label: 'Ace Jump', slug: 'ace-jump' },
					],
				},
				{
					label: 'Flash Messages',
					items: [
						{ label: 'Flash Messages', slug: 'flash-messages' },
					],
				},
				{
					label: 'How-to Guides',
					items: [
						{ label: 'Jujutsu Workflows', slug: 'how-do-i' },
					],
				},
				{
					label: 'Migration Guides',
					items: [
						{ label: 'Migrating to v0.10', slug: 'migrating/v0_10' },
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
