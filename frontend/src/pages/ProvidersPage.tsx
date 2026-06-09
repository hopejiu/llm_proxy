import { useState, useCallback } from "react";
import { ProviderService } from "../../bindings/github.com/wanglejiu/llm-proxy";
import { useProviders } from "../hooks/useProviders";

interface ProviderForm {
  name: string;
  base_url: string;
  api_key: string;
  model: string;
  alias: string;
  auto_suffix: boolean;
  url_suffix: string;
  extra_params: string;
}

const emptyForm: ProviderForm = {
  name: "",
  base_url: "",
  api_key: "",
  model: "",
  alias: "",
  auto_suffix: false,
  url_suffix: "",
  extra_params: "",
};

export default function ProvidersPage() {
  const { providers, loading, error, refresh } = useProviders();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<ProviderForm>(emptyForm);
  const [saving, setSaving] = useState(false);

  const openCreate = () => {
    setForm(emptyForm);
    setEditingId(null);
    setDialogOpen(true);
  };

  const openEdit = (p: any) => {
    setForm({
      name: p.name || "",
      base_url: p.base_url || "",
      api_key: p.api_key || "",
      model: p.model || "",
      alias: p.alias || "",
      auto_suffix: p.auto_suffix || false,
      url_suffix: p.url_suffix || "",
      extra_params: p.extra_params || "",
    });
    setEditingId(p.id);
    setDialogOpen(true);
  };

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      if (editingId) {
        await ProviderService.UpdateProvider(editingId, form);
      } else {
        await ProviderService.CreateProvider(form);
      }
      setDialogOpen(false);
      refresh();
    } catch (e: any) {
      alert("保存失败: " + (e?.message || "未知错误"));
    } finally {
      setSaving(false);
    }
  }, [editingId, form, refresh]);

  const handleDelete = useCallback(
    async (id: number) => {
      if (!confirm("确认删除此 Provider？")) return;
      try {
        await ProviderService.DeleteProvider(id);
        refresh();
      } catch (e: any) {
        alert("删除失败: " + (e?.message || "未知错误"));
      }
    },
    [refresh]
  );

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold text-white">Provider 管理</h1>
        <div className="flex gap-2">
          <button
            onClick={openCreate}
            className="px-3 py-1.5 text-sm bg-blue-600 hover:bg-blue-500 rounded-md transition-colors"
          >
            + 新增
          </button>
          <button
            onClick={() => refresh()}
            className="px-3 py-1.5 text-sm bg-gray-600 hover:bg-gray-500 rounded-md transition-colors"
          >
            刷新
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-700/50 rounded p-3 mb-4 text-sm text-red-300">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-gray-400">加载中...</p>
      ) : providers.length === 0 ? (
        <div className="text-center py-20 text-gray-500">
          <p className="text-lg">暂无 Provider</p>
          <p className="text-sm mt-1">点击「新增」添加第一个 LLM 服务商</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-700/50 text-gray-400">
                <th className="text-left py-3 px-2">名称</th>
                <th className="text-left py-3 px-2">Base URL</th>
                <th className="text-left py-3 px-2">模型</th>
                <th className="text-left py-3 px-2">别名</th>
                <th className="text-right py-3 px-2">操作</th>
              </tr>
            </thead>
            <tbody>
              {providers.map((p) => (
                <tr
                  key={p.id}
                  className="border-b border-gray-800/50 hover:bg-gray-700/20"
                >
                  <td className="py-3 px-2 text-white">{p.name}</td>
                  <td className="py-3 px-2 text-gray-300 max-w-[200px] truncate">
                    {p.base_url}
                  </td>
                  <td className="py-3 px-2 text-gray-300">{p.model || "-"}</td>
                  <td className="py-3 px-2 text-gray-400 text-xs">
                    {p.alias || "-"}
                  </td>
                  <td className="py-3 px-2 text-right">
                    <button
                      onClick={() => openEdit(p)}
                      className="text-blue-400 hover:text-blue-300 mr-3 text-xs"
                    >
                      编辑
                    </button>
                    <button
                      onClick={() => handleDelete(p.id)}
                      className="text-red-400 hover:text-red-300 text-xs"
                    >
                      删除
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Dialog */}
      {dialogOpen && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-[#16213e] border border-gray-600 rounded-lg w-full max-w-lg mx-4 p-6">
            <h2 className="text-lg font-semibold text-white mb-4">
              {editingId ? "编辑 Provider" : "新增 Provider"}
            </h2>
            <div className="space-y-3">
              {(
                [
                  ["name", "名称"],
                  ["base_url", "Base URL"],
                  ["api_key", "API Key"],
                  ["model", "模型"],
                  ["alias", "别名"],
                  ["url_suffix", "URL 后缀"],
                ] as [keyof ProviderForm, string][]
              ).map(([key, label]) => (
                <div key={key}>
                  <label className="block text-xs text-gray-400 mb-1">
                    {label}
                  </label>
                  <input
                    type={key === "api_key" ? "password" : "text"}
                    value={form[key] as string}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, [key]: e.target.value }))
                    }
                    className="w-full bg-[#0f3460] border border-gray-600 rounded px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
              ))}
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="auto_suffix"
                  checked={form.auto_suffix}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, auto_suffix: e.target.checked }))
                  }
                  className="rounded"
                />
                <label htmlFor="auto_suffix" className="text-xs text-gray-400">
                  自动添加 URL 后缀
                </label>
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">
                  额外参数 (JSON)
                </label>
                <textarea
                  value={form.extra_params}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      extra_params: e.target.value,
                    }))
                  }
                  rows={3}
                  className="w-full bg-[#0f3460] border border-gray-600 rounded px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setDialogOpen(false)}
                className="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleSave}
                disabled={saving}
                className="px-4 py-2 text-sm bg-blue-600 hover:bg-blue-500 disabled:opacity-50 rounded-md transition-colors"
              >
                {saving ? "保存中..." : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
