import { Code2 } from "lucide-preact";
import wavePollLogo from "/wave_pool.svg";
import { A, navigate, url } from "../lib/router";
import { api } from "../lib/api";
import { user } from "../lib/session";

const logout = api["DELETE/api/v1/auth/logout"].signal();
const logoutHandler = async (e: Event) => {
  e.preventDefault();
  const refreshToken = localStorage.getItem("refresh_token");
  if (refreshToken) {
    await logout.fetch({ refresh_token: refreshToken });
    if (logout.data) {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("expires_at");
      user.reset();
      navigate({ params: { nav: "login" } });
    }
  }
};
const TabButton = ({ name }: { name: string }) => (
  <A
    params={{ tab: name.toLowerCase().replace(" ", "-") }}
    className={`tab tab-bordered ${
      url.params.tab === name.toLowerCase().replace(" ", "-")
        ? "tab-active text-cyan-400 border-cyan-400 font-semibold"
        : "text-gray-500 font-semibold"
    }`}
  >
    {name}
  </A>
);

export function DeveloperPortal() {
  const { nav, tab } = url.params;
  if (nav !== "dev-portal" || !tab) {
    navigate({ params: { nav: "dev-portal", tab: "api-keys" } });
  }
  return (
    <div className="flex min-h-screen bg-gray-50">
      <aside className="w-60 bg-white border-r border-gray-200">
        <div className="flex items-center gap-2 p-6 border-b border-gray-200">
          <div className="w-8 h-8 bg-gradient-to-br rounded-lg flex items-center justify-center">
            <img className="w-8 h-8" src={wavePollLogo} alt="Wave Pool Logo" />
          </div>
          <span className="text-xl text-gray-900 font-bold">Wave Pool</span>
        </div>

        <nav className="p-4">
          <ul className="space-y-1">
            <li>
              <A
                params={{ nav: "dev-portal" }}
                className={`flex items-center gap-3 px-4 py-3 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors ${
                  url.params.nav === "dev-portal"
                    ? "bg-gray-100 font-semibold"
                    : ""
                }`}
              >
                <Code2 className="w-5 h-5 text-blue-500" />
                <span className="font-medium">Developer Portal</span>
              </A>
            </li>
          </ul>
        </nav>
      </aside>

      <div className="flex-1 flex flex-col">
        <header className="bg-white border-b border-gray-200 px-8 py-4">
          <div className="flex items-center justify-end">
            <div
              role="button"
              onClick={logoutHandler}
              className="flex items-center gap-3 cursor-pointer hover:bg-gray-100 px-3 py-1 rounded-lg transition-colors"
            >
              <div className="text-right">
                <div className="font-semibold text-gray-900">Pengouin Shop</div>
                <div className="text-sm text-gray-500">SN Pingouin Shop</div>
              </div>
              <div className="avatar placeholder">
                <div className="bg-cyan-400 text-white rounded-full w-10 h-10 flex items-center justify-center">
                  <span className="text-lg font-semibold">P</span>
                </div>
              </div>
            </div>
          </div>
        </header>

        <main className="flex-1 p-8">
          <div className="max-w-7xl mx-auto">
            <h1 className="text-4xl font-bold text-gray-900 mb-8">
              Developer Portal
            </h1>

            <div className="tabs tabs-bordered mb-8">
              <TabButton name="API KEYS" />
              <TabButton name="WEBHOOKS" />
            </div>

            <div className="flex justify-end mb-6">
              <A
                params={{ action: "create-api-key" }}
                className="btn btn-sm bg-cyan-400 hover:bg-cyan-500 text-white border-none normal-case"
              >
                Create API Key
              </A>
            </div>

            <div className="bg-white rounded-lg shadow-sm border border-gray-200">
              <div className="overflow-x-auto">
                <table className="table w-full">
                  <thead>
                    <tr className="border-b border-gray-200">
                      <th className="bg-white text-gray-700 font-semibold text-base">
                        key
                      </th>
                      <th className="bg-white text-gray-700 font-semibold text-base">
                        APIs
                      </th>
                      <th className="bg-white text-gray-700 font-semibold text-base text-right">
                        Manage
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr className="border-b border-gray-100 hover:bg-gray-50">
                      <td className="text-gray-900">
                        <span className="font-mono">wave_sn_prod_....●●●●</span>
                      </td>
                      <td className="text-gray-700">Balance API</td>
                      <td className="text-right">
                        <button className="btn btn-sm btn-outline border-cyan-400 text-cyan-400 hover:bg-cyan-50 hover:border-cyan-500 normal-case">
                          Revoke
                        </button>
                      </td>
                    </tr>
                    <tr className="border-b border-gray-100 hover:bg-gray-50">
                      <td className="text-gray-900">
                        <span className="font-mono">wave_sn_prod_....●●●●</span>
                      </td>
                      <td className="text-gray-700">
                        Balance API, Checkout API
                      </td>
                      <td className="text-right">
                        <button className="btn btn-sm btn-outline border-cyan-400 text-cyan-400 hover:bg-cyan-50 hover:border-cyan-500 normal-case">
                          Revoke
                        </button>
                      </td>
                    </tr>
                    <tr className="hover:bg-gray-50">
                      <td className="text-gray-900">
                        <span className="font-mono">wave_sn_test_....●●●●</span>
                      </td>
                      <td className="text-gray-700">
                        Balance API, Checkout API
                      </td>
                      <td className="text-right">
                        <button className="btn btn-sm btn-outline border-cyan-400 text-cyan-400 hover:bg-cyan-50 hover:border-cyan-500 normal-case">
                          Revoke
                        </button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}
