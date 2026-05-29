// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'MCP Go Context Server',
			description: 'Servidor MCP para contexto técnico local y memoria operativa persistente.',
			logo: {
				src: './src/assets/scopweb.webp',
				alt: 'MCP Go Context',
			},
			expressiveCode: {
				themes: ['starlight-dark', 'starlight-light'],
			},
			head: [
				{
					tag: 'link',
					attrs: { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
				},
				{
					tag: 'link',
					attrs: { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: true },
				},
				{
					tag: 'link',
					attrs: {
						rel: 'stylesheet',
						href: 'https://fonts.googleapis.com/css2?family=DM+Sans:ital,opsz,wght@0,9..40,300;0,9..40,400;0,9..40,500;0,9..40,600;1,9..40,400&family=Space+Mono:ital,wght@0,400;0,700;1,400&display=swap',
					},
				},
			],
			customCss: ['./src/styles/custom.css'],
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/scopweb/mcp-go-context' },
			],
			sidebar: [
				{
					label: 'Guías',
					items: [
						{ label: 'Introducción', slug: 'guides/introduction' },
						{ label: 'Flujo de Memoria', slug: 'guides/memory-convergence' },
						{ label: 'Buenas Prácticas', slug: 'guides/best-practices' },
						{ label: 'Dashboard y API HTTP', slug: 'guides/dashboard' },
					],
				},
				{
					label: 'Referencia',
					autogenerate: { directory: 'reference' },
				},
			],
		}),
	],
});