import { defineConfig } from "vite";
import preact from "@preact/preset-vite";
import tailwindcss from "@tailwindcss/vite";
// import dotenv from "dotenv";

// dotenv.config();

const PORT = parseInt(process.env.PORT ?? "8081", 10);
const url = process.env.API_URL ?? "http://localhost:" + PORT;
console.log("Proxying API requests to:", url);


// https://vite.dev/config/
export default defineConfig({
  plugins: [preact(), tailwindcss()],
  server: { proxy: { "/api": url } },
});
