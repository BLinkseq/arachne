// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightUiTweaks from 'starlight-ui-tweaks'
import starlightScrollToTop from 'starlight-scroll-to-top'
import codeblocks from 'starlight-codeblocks';

export default defineConfig({
  site: 'https://blinkseq.github.io/arachne',
  base: '/arachne',
	integrations: [
    starlight({
      customCss: ['./src/style/global.css'],
      title: '0.4',
      logo: { src: '/src/assets/logo.png'},
      favicon: '/favicon.png',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/BLinkseq/arachne' }],
      sidebar: [
        { slug: 'install' },
        { slug: 'usage' },
        { slug: 'rfa_em' },
        { slug: 'standardfmt' },
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
          progressRingColor: '#b288f5',
          borderRadius: '50',
        }),
      ],
		}),
	],
});
