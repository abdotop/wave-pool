import { effect } from "@preact/signals";
import { api } from "./api.ts";
import { url } from "./router.tsx";

export const user = api["GET/api/v1/me"].signal();
const refresh = api["POST/api/v1/auth/refresh"].signal();

// Helper pour calculer l'expiration absolue
function setTokens(
  { access_token, refresh_token, expires_in }: {
    access_token: string;
    refresh_token: string;
    expires_in: number;
  },
) {
  localStorage.setItem("access_token", access_token);
  localStorage.setItem("refresh_token", refresh_token);
  // expires_at = timestamp en ms
  const expires_at = Date.now() + expires_in * 1000 - 5000; // marge de 5s
  localStorage.setItem("expires_at", expires_at.toString());
}

// Rafraîchir le token si besoin
async function ensureValidToken() {
  const token = localStorage.getItem("access_token");
  const refreshToken = localStorage.getItem("refresh_token");
  const expiresAt = localStorage.getItem("expires_at");

  if (token && expiresAt && Date.now() < parseInt(expiresAt)) {
    return token;
  }
  if (refreshToken) {
    await refresh.fetch({ refresh_token: refreshToken });
    if (refresh.data) {
      setTokens(refresh.data);
      return refresh.data.access_token;
    } else {
      // Refresh échoué, tout effacer
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("expires_at");
      user.reset();
      return null;
    }
  }
  // Pas de token valide
  user.reset();
  return null;
}

// Synchroniser l'utilisateur à chaque navigation ou changement de token
effect(() => {
  const { path: _ } = url;
  ensureValidToken().then((token) => {
    if (token) {
      user.fetch(undefined, {
        headers: { Authorization: `Bearer ${token}` },
      });
    } else {
      user.reset();
    }
  });
});
