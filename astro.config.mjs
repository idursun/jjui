// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import catppuccin from "@catppuccin/starlight";

/**
 * Rewrites hard-coded /jjui/ paths to the versioned base at build time.
 * Runs as a Vite plugin so it covers both frontmatter YAML and markdown body.
 * Only active when DOCS_BASE differs from the default (/jjui).
 */
function vitePluginRebaseLinks() {
	const newBase = (process.env.DOCS_BASE ?? '/jjui').replace(/\/$/, '');
	if (newBase === '/jjui') return { name: 'rebase-links' };
	return {
		name: 'rebase-links',
		transform(code, id) {
			if (!/\.mdx?$/.test(id)) return null;
			// Match /jjui/ only when preceded by a quote, paren, or whitespace
			// so that external URLs like https://github.com/idursun/jjui/... are untouched.
			const replaced = code.replace(/(["'(\s])(\/jjui)(\/)/g, (_, before, _base, slash) => {
				return before + newBase + slash;
			});
			return replaced !== code ? { code: replaced, map: null } : null;
		},
	};
}

// https://astro.build/config
export default defineConfig({
	site: 'https://idursun.github.io',
	base: process.env.DOCS_BASE ?? '/jjui',
	vite: {
		plugins: [vitePluginRebaseLinks()],
	},
	integrations: [
		starlight({
			title: 'jjui',
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
						{
							label: 'Experimental Features',
							items: [
								{ label: 'Lua Custom Commands', slug: 'features/custom-commands-lua' },
							],
						},
						{
							label: 'Deprecated Features',
							items: [
								{ label: 'Leader Key', slug: 'customization/leader-key' },
							],
						},
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
					label: 'Migration Guides',
					items: [
						{ label: 'Migrating to v0.10', slug: 'migrating/v0_10', },
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
				'./src/styles/custom.css'
			],
		}),
	],
});
