/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  corePlugins: {
    preflight: false,
  },
  theme: {
    extend: {
      colors: {
        clinic: {
          primary: '#7a0000',
          primaryDark: '#560000',
          accent: '#ffdd91',
          background: '#f6f7f8',
          surface: '#ffffff',
          border: '#ead0cb',
          text: '#1f1a1a',
          muted: '#6f5f5f',
        },
      },
      boxShadow: {
        'clinic-soft': '0 18px 40px rgba(31, 26, 26, 0.08)',
      },
    },
  },
  plugins: [],
};
