// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import catppuccin from "@catppuccin/starlight";

// https://astro.build/config
export default defineConfig({
	site: 'https://idursun.github.io',
	base: '/jjui',
	integrations: [
		starlight({
			title: 'jjui',
			// NOTE: icons for top-level are at ./src/components/Sidebar.astro#getIcon
			sidebar: [
				{
					label: 'Jujutsu UI',
					items: [
						{ label: 'Overview', slug: 'overview', },
						{ label: 'Showcase', slug: 'showcase', },
						{ label: 'Install', slug: 'install', },
					],
				},
				{
					label: 'Core Features',
					items: [
						{ label: 'Revisions', slug: 'features/revisions' },
						{ label: 'Preview', slug: 'features/preview' },
						{ label: 'Details', slug: 'features/details' },
						{ label: 'Oplog', slug: 'features/oplog' },
						{ label: 'Command Execution', slug: 'features/command-execution' },
						{ label: 'Custom Commands', slug: 'features/custom-commands' },
						{ label: 'Lua Scripting', slug: 'features/lua-scripting' },
						{ label: 'Fuzzy File Finder', slug: 'features/fuzzy-file-finder' },
						{ label: 'Ace Jump', slug: 'features/ace-jump' },
						{ label: 'Exec', slug: 'features/exec' },
						{ label: 'Flash Messages', slug: 'features/flash-messages' },
						{ label: 'Bookmarks', slug: 'features/bookmarks' },
						{ label: 'Git', slug: 'features/git' },
					],
				},
				{
					label: 'Revision Operations',
					items: [
						{ label: 'Rebase', slug: 'operations/rebase' },
						{ label: 'Absorb', slug: 'operations/absorb' },
						{ label: 'Abandon', slug: 'operations/abandon' },
						{ label: 'Duplicate', slug: 'operations/duplicate' },
						{ label: 'Squash', slug: 'operations/squash' },
						{ label: 'Evolog', slug: 'operations/evolog' },
						{ label: 'Inline Describe', slug: 'operations/inline-describe' },
					],
				},
				{
					label: 'How-to Guides',
					items: [
						{ label: 'Jujutsu Workflows', slug: 'how-do-i', },
					],
				},
				{
					label: 'Customization',
					items: [
						{ label: 'Configuration', slug: 'customization/configuration' },
						{ label: 'Command Line Options', slug: 'customization/command-line-options' },
						{ label: 'Themes', slug: 'customization/themes' },
					],
				},
				{
					label: 'Experimental Features',
					items: [
						{ label: 'Lua Custom Commands', slug: 'features/custom-commands-lua' },
						{ label: 'Tracing', slug: 'customization/tracing' },
					],
				},
				{
					label: 'Deprecated Features',
					items: [
						{ label: 'Leader Key', slug: 'customization/leader-key' },
					],
				},
				{
					label: 'Migration Guides',
					items: [
  				      { label: 'Migrating to v0.10', slug: 'migrating/v0_10', },
					],
				},
			],
			components: {
				Sidebar: './src/components/Sidebar.astro',
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
				'./src/styles/custom.css'
			],
		}),
	],
});
