/** @type {import('tailwindcss').Config} */
// module.exports = {
//     content: ["./src/**/*.{html,js}"],
//     theme: {
//         extend: {},
//     },
//     plugins: [],
// }

module.exports = {
    purge: ["./src/**/*.html", "./src/**/*.js"],
    darkMode: false, // or 'media' or 'class'
    theme: {
        extend: {},
    },
    variants: {},
    plugins: [],
}