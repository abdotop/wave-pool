import { api } from "../lib/api";
import wavePollLogo from "/wave_pool.svg";
import senegalFlag from "../assets/flags/senegal.svg";
import { Lock } from "lucide-preact";
import { user } from "../lib/session";

const register = api["POST/api/v1/auth"].signal();

const handleSubmit = async (e: Event) => {
  e.preventDefault();
  const form = e.target as HTMLFormElement;
  const formData = new FormData(form);
  const countryCode = formData.get("countryCode") as string;
  const phone = formData.get("phone") as string;
  const pin = formData.get("pin") as string;

  await register.fetch({
    phone: countryCode + phone,
    pin,
  });
  if (register.data) {
    if (typeof register.data === "string") {
      register.error =  Error(register.data);
      form.reset();
      return;
    }
    localStorage.setItem("access_token", register.data.access_token);
    localStorage.setItem("refresh_token", register.data.refresh_token);
    localStorage.setItem("expires_in", register.data.expires_in.toString());
    await user.fetch(undefined, {
      headers: {
        Authorization: `Bearer ${register.data.access_token}`,
      },
    });
    form.reset();
  }
};

export function LoginPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-base-100">
      <div className="w-full max-w-md px-6">
        <div className="flex justify-center mb-8">
          <div className="flex items-center gap-2">
            <div className="w-12 h-12 bg-[#00D9FF] rounded-lg flex items-center justify-center">
              <img
                className="w-12 h-12"
                src={wavePollLogo}
                alt="Wave Pool Logo"
              />
            </div>
            <span className="text-2xl font-semibold text-base-content">
              Wave Pool
            </span>
          </div>
        </div>
        <div>
          {register.error && (
            <p className="text-red-500">{register.error.message}</p>
          )}
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="flex gap-2">
            <select
              name="countryCode"
              className="select select-bordered w-32"
              defaultValue="+221"
            >
              <option value="+221">
                <img src={senegalFlag} alt="Senegal Flag" className="inline w-5 h-3 mr-2" />
                +221
              </option>
            </select>
            <div className="flex-1">
              <input
                type="tel"
                name="phone"
                placeholder="Numéro de Téléphone *"
                className="input input-bordered w-full"
                required
              />
            </div>
          </div>
          <div className="form-control">
            <label className="label">
              <span className="label-text">Code PIN (4 chiffres)</span>
            </label>
            <div className="relative">
              <input
                type="password"
                name="pin"
                placeholder="••••"
                className="input input-bordered w-full text-center text-2xl tracking-[0.5em] font-bold"
                maxLength={4}
                pattern="[0-9]{4}"
                required
              />
              <Lock className="absolute right-3 top-1/2 -translate-y-1/2 w-5 h-5 text-base-content/40" />
            </div>
          </div>
          <button
            type="submit"
            disabled={!register.pending === false}
            className="btn btn-info w-full text-white text-base"
          >
            {(!register.pending === false || !user.pending === false)
              ? "Connexion..."
              : "Se Connecter"}
          </button>
        </form>
        <div className="text-center mt-8">
          <a href="#" className="text-info text-sm hover:underline">
            Avis de Confidentialité
          </a>
        </div>
      </div>
    </div>
  );
}
