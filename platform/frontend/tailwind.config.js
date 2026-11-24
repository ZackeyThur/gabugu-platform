/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // 浅色主题（2026-11-22 起）：以 #F2E8DB 米色为画布的暖纸色系。
        // pearl-* 是浅色专用背景；gray-*、primary-*、白色背景改读 CSS 变量，
        // 变量在 style.css 里定义：浅色为暖色值，.dark 与固定深色页恢复 Tailwind 原值。
        pearl: {
          bg: '#f2e8db',
          surface: '#fbf7f0',
          raised: '#e8dbc8',
          line: '#dacbb4'
        },
        // 赤陶：与米色同色相的暖色点缀（徽章、引导、次级强调）。
        clay: {
          50: '#faf1ea',
          100: '#f3e3d8',
          200: '#e6c9b6',
          300: '#d6a98f',
          400: '#c4825f',
          500: '#b85d3b',
          600: '#a5532f',
          700: '#8a4426',
          800: '#6e3720',
          900: '#522a19'
        },
        gray: Object.fromEntries(
          [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950].map((n) => [
            n,
            `rgb(var(--gray-${n}) / <alpha-value>)`
          ])
        ),
        // GABUGU — electric blue, from the approved brand identity.
        primary: Object.fromEntries(
          [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950].map((n) => [
            n,
            `rgb(var(--primary-${n}) / <alpha-value>)`
          ])
        ),
        ...Object.fromEntries(
          ['emerald', 'green', 'amber', 'yellow', 'red', 'rose', 'orange'].map((c) => [
            c,
            Object.fromEntries(
              [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950].map((n) => [
                n,
                `rgb(var(--${c}-${n}) / <alpha-value>)`
              ])
            )
          ])
        ),
        // 直接写 blue-* 的 800+ 处同样跟随主题：浅色收拢为黛蓝，深色还原 Tailwind 原值。
        blue: Object.fromEntries(
          [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950].map((n) => [
            n,
            `rgb(var(--blue-${n}) / <alpha-value>)`
          ])
        ),
        // 辅助色 - 深蓝灰
        accent: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          600: '#475569',
          700: '#334155',
          800: '#1e293b',
          900: '#0f172a',
          950: '#020617'
        },
        // 深色模式背景（读 CSS 变量：工作台 .dark 使用同源暖炭黑，.gabugu-auth/.gabugu-landing 保持独立深黑舞台）
        dark: Object.fromEntries(
          [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950].map((n) => [
            n,
            `rgb(var(--dark-${n}) / <alpha-value>)`
          ])
        )
      },
      ringOffsetColor: { DEFAULT: 'rgb(var(--c-white))', white: 'rgb(var(--c-white))' },
      backgroundColor: { white: 'rgb(var(--c-white) / <alpha-value>)' },
      gradientColorStops: { white: 'rgb(var(--c-white) / <alpha-value>)' },
      fontFamily: {
        sans: [
          'Inter',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: '0 8px 32px rgb(var(--shadow) / 0.08), inset 0 1px 0 rgb(255 255 255 / 0.5)',
        'glass-sm': '0 4px 16px rgb(var(--shadow) / 0.06), inset 0 1px 0 rgb(255 255 255 / 0.45)',
        glow: '0 0 20px rgb(var(--primary-500) / 0.25)',
        'glow-lg': '0 0 40px rgb(var(--primary-500) / 0.35)',
        card: '0 1px 2px rgb(var(--shadow) / 0.05), 0 4px 16px -4px rgb(var(--shadow) / 0.06), inset 0 1px 0 var(--surface-rim)',
        'card-hover': '0 6px 24px -6px rgb(var(--shadow) / 0.11), 0 2px 6px -1px rgb(var(--shadow) / 0.06), inset 0 1px 0 var(--surface-rim)',
        'card-elevated': '0 10px 32px -10px rgb(var(--shadow) / 0.14), 0 2px 8px -2px rgb(var(--shadow) / 0.06), inset 0 1px 0 var(--surface-rim)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.1)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary':
          'linear-gradient(135deg, rgb(var(--primary-500)) 0%, rgb(var(--primary-600)) 100%)',
        'gradient-dark': 'linear-gradient(135deg, rgb(var(--dark-800)) 0%, rgb(var(--dark-900)) 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgb(var(--mesh-a)) 0px, transparent 50%), radial-gradient(at 80% 0%, rgb(var(--mesh-b)) 0px, transparent 50%), radial-gradient(at 0% 50%, rgb(var(--mesh-c)) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgb(var(--primary-500) / 0.25)' },
          '100%': { boxShadow: '0 0 30px rgb(var(--primary-500) / 0.4)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
