// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightUiTweaks from 'starlight-ui-tweaks'
import starlightScrollToTop from 'starlight-scroll-to-top'
import codeblocks from 'starlight-codeblocks';
import mermaid from 'astro-mermaid';

export default defineConfig({
  site: 'https://blinkseq.github.io/arachne',
  base: '/arachne',
	integrations: [
    mermaid(),
    starlight({
      customCss: ['./src/style/global.css'],
      title: '0.4',
      logo: {
        light: '/src/assets/logo.png',
        dark: '/src/assets/logo_dark.png',
      },
      favicon: '/favicon.png',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/BLinkseq/arachne' }],
			lastUpdated: true,
      pagination: false,
      editLink: {
        baseUrl: 'https://github.com/blinkseq/arachne/edit/main/docs/',
      },
      sidebar: [
        { slug: 'install' },
        { slug: 'rfa_em' },
        {
          label: 'Usage',
          items: [{ autogenerate: { directory: 'usage' } }],
        },
      ],
      plugins: [
        codeblocks(),
        starlightUiTweaks({
        navbarLinks: [
           { label: "BLink-seq", href: "https://blinkseq.github.io"},
        ],
      }),
        starlightScrollToTop({
          showProgressRing: true,
          progressRingColor: '#222124',
          borderRadius: '50',
        }),
      ],
		}),
	],
});
