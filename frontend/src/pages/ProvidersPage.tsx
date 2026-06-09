import { useState, useCallback, useMemo, useRef } from "react";
import { ProviderAPI } from "../services";
import { useProviders } from "../hooks/useProviders";
import { useToast } from "../components/Toast";
import LoadingSpinner from "../components/LoadingSpinner";
import ErrorBanner from "../components/ErrorBanner";
import Modal from "../components/Modal";
import ConfirmDialog from "../components/ConfirmDialog";
import { useHotkey } from "../hooks/useHotkey";

/* ---------- types ---------- */

interface ModelEntry {
  name: string;
  aliases: string[];
  extra_params: string;
  input_price: number;
  output_price: number;
  cache_price: number;
}

interface ProviderForm {
  name: string;
  base_url: string;
  api_key: string;
  auto_suffix: boolean;
  url_suffix: string;
  models: ModelEntry[];
}

interface FormErrors {
  name?: string;
  base_url?: string;
  api_key?: string;
  models?: string;
}

const emptyModel = (): ModelEntry => ({ name: "", aliases: [], extra_params: "", input_price: 0, output_price: 0, cache_price: 0 });

const emptyForm: ProviderForm = {
  name: "", base_url: "", api_key: "",
  auto_suffix: false, url_suffix: "",
  models: [],
};

/* ---------- helpers ---------- */

function validate(form: ProviderForm): FormErrors {
  const e: FormErrors = {};
  if (!form.name.trim()) e.name = "名称不能为空";
  if (!form.base_url.trim()) e.base_url = "Base URL 不能为空";
  else if (!/^https?:\/\//i.test(form.base_url)) e.base_url = "URL 需以 http:// 或 https:// 开头";
  if (!form.api_key.trim()) e.api_key = "API Key 不能为空";
  if (form.models.length === 0) e.models = "至少需要一个模型";
  else if (!form.models.some((m) => m.name.trim())) e.models = "每个模型的上游名称不能为空";
  return e;
}

/** Serialize models to JSON string for the backend `models` field */
function serializeModels(models: ModelEntry[]): string {
  return JSON.stringify(models);
}

/** Ensure the form's models are serializable (strip empty entries) */
function cleanModels(models: ModelEntry[]): ModelEntry[] {
  return models.filter((m) => m.name.trim() !== "");
}

/* ---------- TagInput (aliases) ---------- */

function TagInput({
  tags,
  onChange,
  placeholder,
}: {
  tags: string[];
  onChange: (next: string[]) => void;
  placeholder?: string;
}) {
  const [input, setInput] = useState("");

  const addTag = () => {
    const val = input.trim();
    if (val && !tags.includes(val)) {
      onChange([...tags, val]);
    }
    setInput("");
  };

  const removeTag = (i: number) => {
    onChange(tags.filter((_, idx) => idx !== i));
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      addTag();
    }
    if (e.key === "Backspace" && input === "" && tags.length > 0) {
      removeTag(tags.length - 1);
    }
  };

  return (
    <div
      className="flex flex-wrap gap-1 items-center px-2 py-1.5 border border-[#EDE9FE] rounded-lg bg-white min-h-[36px] cursor-text focus-within:ring-2 focus-within:ring-brand-600/20 focus-within:border-brand-600/40 transition-all"
      onClick={() => document.getElementById(`tag-input-${Math.random()}`)?.focus()}
    >
      {tags.map((t, i) => (
        <span
          key={i}
          className="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full bg-brand-50 text-brand-700 border border-brand-200"
        >
          {t}
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); removeTag(i); }}
            className="text-brand-400 hover:text-brand-700 leading-none"
            aria-label={`删除别名 ${t}`}
          >
            ×
          </button>
        </span>
      ))}
      <input
        type="text"
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={addTag}
        className="flex-1 min-w-[80px] outline-none text-xs bg-transparent border-none p-0"
        placeholder={tags.length === 0 ? (placeholder || "输入后回车添加") : ""}
      />
    </div>
  );
}

/* ---------- ModelCard ---------- */

function ModelCard({
  entry,
  index,
  onChange,
  onRemove,
}: {
  entry: ModelEntry;
  index: number;
  onChange: (idx: number, next: ModelEntry) => void;
  onRemove: (idx: number) => void;
}) {
  const [collapsed, setCollapsed] = useState(true);

  const update = (partial: Partial<ModelEntry>) => {
    onChange(index, { ...entry, ...partial });
  };

  return (
    <div className="border border-[#EDE9FE] rounded-xl bg-white overflow-hidden">
      {/* Header row */}
      <div className="flex items-center gap-2 px-3 py-2.5 bg-[#FAF5FF]">
        <span className="w-5 h-5 rounded-full bg-brand-600 text-white text-[11px] font-bold flex items-center justify-center shrink-0">
          {index + 1}
        </span>
        <input
          type="text"
          value={entry.name}
          onChange={(e) => update({ name: e.target.value })}
          placeholder="上游模型名（如 gpt-4）"
          className="flex-1 text-sm font-medium bg-white border border-[#EDE9FE] rounded-lg px-2.5 py-1.5 outline-none focus:ring-2 focus:ring-brand-600/20 focus:border-brand-600/40 transition-all placeholder:text-[#C4BDD5]"
        />
        <button
          type="button"
          onClick={() => setCollapsed(!collapsed)}
          className="text-[#9C94B0] hover:text-[#6B6580] p-1 transition-colors"
          aria-label={collapsed ? "展开扩展参数" : "收起扩展参数"}
        >
          <svg
            className={`w-4 h-4 transition-transform ${collapsed ? "" : "rotate-180"}`}
            fill="none" stroke="currentColor" viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>
        <button
          type="button"
          onClick={() => onRemove(index)}
          className="text-red-300 hover:text-red-500 p-1 transition-colors"
          aria-label={`删除模型 ${entry.name || `#${index + 1}`}`}
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        </button>
      </div>

      {/* Aliases row (always visible) */}
      <div className="px-3 py-2">
        <label className="text-[11px] font-medium text-[#9C94B0] mb-1 block">
          别名（客户端可用的模型名，多个别名用回车分隔）
        </label>
        <TagInput
          tags={entry.aliases}
          onChange={(next) => update({ aliases: next })}
          placeholder="输入别名后回车"
        />
      </div>

      {/* Pricing row (always visible) */}
      <div className="px-3 py-2 border-t border-[#F0EBF5]">
        <label className="text-[11px] font-medium text-[#9C94B0] mb-1.5 block">
          计费价格（元/百万 token，0=未设置）
        </label>
        <div className="grid grid-cols-3 gap-2">
          {[
            { key: "input_price" as const, label: "输入价" },
            { key: "output_price" as const, label: "输出价" },
            { key: "cache_price" as const, label: "缓存价" },
          ].map(({ key, label }) => (
            <div key={key}>
              <span className="text-[10px] text-[#9C94B0] mb-0.5 block">{label}</span>
              <input
                type="text"
                inputMode="decimal"
                defaultValue={entry[key] > 0 ? String(entry[key]) : ""}
                onBlur={(e) => {
                  const raw = e.target.value.trim();
                  const num = raw === "" ? 0 : parseFloat(raw);
                  update({ [key]: isNaN(num) ? 0 : num });
                }}
                onKeyDown={(e) => {
                  if (e.key === "Enter") (e.target as HTMLInputElement).blur();
                }}
                className="input-field text-xs w-full"
                placeholder="0.00"
              />
            </div>
          ))}
        </div>
      </div>

      {/* Extra params (collapsible) */}
      {!collapsed && (
        <div className="px-3 pb-3">
          <label className="text-[11px] font-medium text-[#9C94B0] mb-1 block">
            扩展参数 (JSON)
          </label>
          <textarea
            value={entry.extra_params}
            onChange={(e) => update({ extra_params: e.target.value })}
            rows={2}
            className="input-field font-mono text-xs w-full"
            placeholder='{"temperature": 0.7, "max_tokens": 4096}'
          />
        </div>
      )}
    </div>
  );
}

/* ---------- ProviderFormFields ---------- */

const baseFields: [keyof ProviderForm, string, string][] = [
  ["name", "名称", "text"],
  ["base_url", "Base URL", "text"],
  ["api_key", "API Key", "text"],
  ["url_suffix", "URL 后缀", "text"],
];

/* ---------- Main page ---------- */

export default function ProvidersPage() {
  const { providers, loading, error, refresh } = useProviders();
  const { toast } = useToast();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<ProviderForm>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [errors, setErrors] = useState<FormErrors>({});
  const [touched, setTouched] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null);
  const formId = useRef(0); // force re-mount for editor state reset

  // Test connection
  const [testStatus, setTestStatus] = useState<{
    loading: boolean;
    ok?: boolean;
    message?: string;
  }>({ loading: false });
  const [testModel, setTestModel] = useState<string>("");

  // Fetch models from API
  const [fetchOpen, setFetchOpen] = useState(false);
  const [fetchModels, setFetchModels] = useState<string[]>([]);
  const [fetchSearch, setFetchSearch] = useState("");
  const [fetchLoading, setFetchLoading] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [fetchSelected, setFetchSelected] = useState<Set<string>>(new Set());

  // Ctrl+N new provider, Ctrl+R refresh
  useHotkey("Ctrl+N", useCallback(() => { openCreate(); }, []));
  useHotkey("Ctrl+R", useCallback(() => refresh(), [refresh]));

  const openCreate = () => {
    formId.current++;
    setForm(emptyForm);
    setEditingId(null);
    setErrors({});
    setTouched(new Set());
    setTestStatus({ loading: false });
    setTestModel("");
    setDialogOpen(true);
  };

  const openEdit = (p: any) => {
    let models: ModelEntry[] = [];
    try {
      models = JSON.parse(p.models || "[]");
    } catch { models = []; }
    formId.current++;
    setForm({
      name: p.name || "",
      base_url: p.base_url || "",
      api_key: p.api_key || "",
      auto_suffix: p.auto_suffix || false,
      url_suffix: p.url_suffix || "",
      models,
    });
    setEditingId(p.id);
    setErrors({});
    setTouched(new Set());
    setTestStatus({ loading: false });
    setTestModel(models.length > 0 ? models[0].name : "");
    setDialogOpen(true);
  };

  const handleFieldChange = (key: keyof ProviderForm, value: string | boolean) => {
    const next = { ...form, [key]: value };
    setForm(next);
    setTouched((prev) => new Set(prev).add(key));
    setErrors(validate(next));
  };

  const handleModelChange = (idx: number, entry: ModelEntry) => {
    const next = [...form.models];
    next[idx] = entry;
    const updated = { ...form, models: next };
    setForm(updated);
    setTouched((prev) => new Set(prev).add("models"));
    setErrors(validate(updated));
  };

  const addModel = () => {
    const next = [...form.models, emptyModel()];
    setForm({ ...form, models: next });
    setTouched((prev) => new Set(prev).add("models"));
  };

  const removeModel = (idx: number) => {
    const next = form.models.filter((_, i) => i !== idx);
    setForm({ ...form, models: next });
    setTouched((prev) => new Set(prev).add("models"));
  };

  // Fetch models from upstream API
  const openFetchModels = async () => {
    if (!form.base_url.trim()) {
      toast("info", "请先填写 Base URL");
      return;
    }
    setFetchLoading(true);
    setFetchError(null);
    setFetchModels([]);
    setFetchSearch("");
    setFetchSelected(new Set());
    setFetchOpen(true);
    try {
      const list = await ProviderAPI.fetchModels(form.base_url, form.api_key);
      setFetchModels(list);
    } catch (err: any) {
      setFetchError(err?.message || "查询失败");
    } finally {
      setFetchLoading(false);
    }
  };

  const toggleFetchModel = (name: string) => {
    setFetchSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const addSelectedModels = () => {
    if (fetchSelected.size === 0) return;
    const existingNames = new Set(form.models.map((m) => m.name));
    const toAdd: ModelEntry[] = [];
    fetchSelected.forEach((name) => {
      if (!existingNames.has(name)) {
        toAdd.push({ name, aliases: [], extra_params: "", input_price: 0, output_price: 0, cache_price: 0 });
      }
    });
    if (toAdd.length === 0) {
      toast("info", "所选模型已在列表中");
      return;
    }
    setForm({ ...form, models: [...form.models, ...toAdd] });
    setTouched((prev) => new Set(prev).add("models"));
    setFetchOpen(false);
    // Auto-select first model for testing if none selected
    if (!testModel.trim() && toAdd.length > 0) {
      setTestModel(toAdd[0].name);
    }
    toast("success", `已添加 ${toAdd.length} 个模型`);
  };

  // Test connection
  const handleTestConnection = async () => {
    if (!form.base_url.trim()) {
      setTestStatus({ loading: false, ok: false, message: "请先填写 Base URL" });
      return;
    }
    if (!form.api_key.trim()) {
      setTestStatus({ loading: false, ok: false, message: "请先填写 API Key" });
      return;
    }
    if (!testModel.trim()) {
      setTestStatus({ loading: false, ok: false, message: "请先选择要测试的模型" });
      return;
    }
    setTestStatus({ loading: true });
    try {
      const msg = await ProviderAPI.testConnection(
        form.base_url,
        form.api_key,
        testModel,
        form.url_suffix,
        form.auto_suffix,
      );
      setTestStatus({ loading: false, ok: msg.startsWith("✅"), message: msg });
    } catch (err: any) {
      setTestStatus({ loading: false, ok: false, message: err?.message || "连接测试失败" });
    }
  };

  const handleSave = useCallback(async () => {
    const cleaned = cleanModels(form.models);
    const formWithClean = { ...form, models: cleaned };
    const e = validate(formWithClean);
    setErrors(e);
    setTouched(new Set(Object.keys(e)));
    if (Object.keys(e).length > 0) return;

    setSaving(true);
    setActionError(null);
    try {
      const payload = {
        ...formWithClean,
        models: serializeModels(cleaned),
      };
      if (editingId) {
        await ProviderAPI.updateProvider(editingId, payload);
        toast("success", "Provider 已更新");
      } else {
        await ProviderAPI.createProvider(payload);
        toast("success", "Provider 已创建");
      }
      setDialogOpen(false);
      refresh();
    } catch (err: any) {
      setActionError("保存失败: " + (err?.message || "未知错误"));
    } finally {
      setSaving(false);
    }
  }, [form, editingId, refresh, toast]);

  const handleDelete = useCallback(async () => {
    if (deleteTarget == null) return;
    try {
      await ProviderAPI.deleteProvider(deleteTarget);
      toast("success", "Provider 已删除");
      refresh();
    } catch (e: any) {
      setActionError("删除失败: " + (e?.message || "未知错误"));
    } finally {
      setDeleteTarget(null);
    }
  }, [deleteTarget, refresh, toast]);

  // Search across all model names and aliases
  const filtered = useMemo(() => {
    if (!search) return providers;
    const q = search.toLowerCase();
    return providers.filter((p: any) => {
      if ((p.name || "").toLowerCase().includes(q)) return true;
      try {
        const models: ModelEntry[] = JSON.parse(p.models || "[]");
        for (const m of models) {
          if (m.name.toLowerCase().includes(q)) return true;
          for (const a of m.aliases) {
            if (a.toLowerCase().includes(q)) return true;
          }
        }
      } catch {
        // ignore
      }
      return false;
    });
  }, [providers, search]);

  // Build model display strings for table
  const modelDisplay = (p: any): string => {
    try {
      const models: ModelEntry[] = JSON.parse(p.models || "[]");
      return models.map((m) => {
        const parts = [m.name];
        if (m.aliases.length > 0) parts.push(`(${m.aliases.join(", ")})`);
        return parts.join(" ");
      }).join(" | ");
    } catch {
      return "-";
    }
  };

  return (
    <div className="page-container">
      <div className="page-header">
        <h1 className="page-title">Provider 管理</h1>
        <div className="flex gap-2">
          <button onClick={() => refresh()} className="btn-secondary" aria-label="刷新">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            刷新
          </button>
          <button onClick={openCreate} className="btn-primary" aria-label="新增 Provider (Ctrl+N)">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            新增
          </button>
        </div>
      </div>

      <ErrorBanner error={error} />
      <ErrorBanner error={actionError} onClose={() => setActionError(null)} />

      {/* Search */}
      <div className="relative mb-4 max-w-xs">
        <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#9C94B0]" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索 Provider 名称或模型..." className="input-field pl-8 text-xs" aria-label="搜索 Provider" />
      </div>

      <div className="section-card">
        {loading ? (
          <LoadingSpinner />
        ) : filtered.length === 0 && !search ? (
          <div className="text-center py-12">
            <div className="w-12 h-12 rounded-full bg-brand-50 flex items-center justify-center mx-auto mb-4">
              <svg className="w-6 h-6 text-brand-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
              </svg>
            </div>
            <p className="text-base font-medium text-[#1E1B2E]">开始使用 LLM Proxy</p>
            <ol className="text-left max-w-xs mx-auto mt-4 space-y-2 text-sm">
              <li className="flex gap-2 items-center"><span className="w-5 h-5 rounded-full bg-brand-600 text-white text-xs flex items-center justify-center shrink-0">1</span> 点击「新增」添加第一个 LLM 服务商</li>
              <li className="flex gap-2 items-center"><span className="w-5 h-5 rounded-full bg-brand-600 text-white text-xs flex items-center justify-center shrink-0">2</span> 在「设置」中启动代理服务</li>
              <li className="flex gap-2 items-center"><span className="w-5 h-5 rounded-full bg-slate-200 text-slate-500 text-xs flex items-center justify-center shrink-0">3</span> 配置客户端指向 localhost:8888</li>
            </ol>
          </div>
        ) : (
          <div className="table-wrap">
            <table className="table-base" role="table" aria-label="Provider 列表">
              <thead>
                <tr className="border-b border-[#F0EBF5]">
                  <th className="table-th">名称</th>
                  <th className="table-th">Base URL</th>
                  <th className="table-th">模型</th>
                  <th className="table-th text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 ? (
                  <tr><td colSpan={4} className="text-center py-12 text-[#9C94B0] text-xs">未找到匹配的 Provider</td></tr>
                ) : (
                  filtered.map((p: any) => (
                    <tr key={p.id} className="table-tr">
                      <td className="table-td font-medium">{p.name}</td>
                      <td className="table-td text-[#6B6580] max-w-[200px] truncate font-mono text-xs">{p.base_url}</td>
                      <td className="table-td text-[#6B6580] text-xs max-w-[300px]">
                        {p.models ? (
                          <span className="truncate block" title={modelDisplay(p)}>{modelDisplay(p)}</span>
                        ) : (
                          <span className="text-[#9C94B0]">-</span>
                        )}
                      </td>
                      <td className="py-3 px-3 text-right">
                        <button onClick={() => openEdit(p)} className="text-brand-600 hover:text-brand-700 text-xs font-medium mr-4 transition-colors" aria-label={`编辑 ${p.name}`}>编辑</button>
                        <button onClick={() => setDeleteTarget(p.id)} className="text-red-500 hover:text-red-600 text-xs font-medium transition-colors" aria-label={`删除 ${p.name}`}>删除</button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create/Edit Modal */}
      <Modal
        key={formId.current}
        open={dialogOpen}
        onClose={() => { if (!saving) setDialogOpen(false); }}
        title={editingId ? "编辑 Provider" : "新增 Provider"}
        footer={
          <>
            <button onClick={() => setDialogOpen(false)} className="btn-secondary">取消</button>
            <button onClick={handleSave} disabled={saving || Object.keys(errors).length > 0} className="btn-primary">
              {saving ? (
                <span className="flex items-center gap-1.5">
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24" aria-hidden="true">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  保存中...
                </span>
              ) : "保存"}
            </button>
          </>
        }
      >
        <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-1">
          {/* Base fields */}
          {baseFields.map(([key, label, type]) => (
            <div key={key}>
              <label className="form-label">
                {label}
                {["name", "base_url", "api_key"].includes(key) && (
                  <span className="text-red-400 ml-0.5">*</span>
                )}
              </label>
              <input
                type={type}
                value={form[key] as string}
                onChange={(e) => handleFieldChange(key, e.target.value)}
                className={`input-field ${touched.has(key) && errors[key as keyof FormErrors] ? "border-red-300 bg-red-50" : ""}`}
                placeholder={label}
                aria-invalid={!!(touched.has(key) && errors[key as keyof FormErrors])}
                aria-describedby={errors[key as keyof FormErrors] ? `${key}-err` : undefined}
              />
              {touched.has(key) && errors[key as keyof FormErrors] && (
                <p id={`${key}-err`} className="text-xs text-red-500 mt-0.5" role="alert">{errors[key as keyof FormErrors]}</p>
              )}
            </div>
          ))}

          {/* Auto suffix checkbox */}
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              id="auto_suffix"
              checked={form.auto_suffix}
              onChange={(e) => handleFieldChange("auto_suffix", e.target.checked)}
              className="w-4 h-4 rounded border-[#EDE9FE] text-brand-600 focus:ring-brand-600/20"
            />
            <label htmlFor="auto_suffix" className="text-sm text-[#6B6580] cursor-pointer">自动添加 URL 后缀</label>
          </label>

          {/* Models section */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="form-label mb-0">
                模型配置
                <span className="text-red-400 ml-0.5">*</span>
              </label>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={openFetchModels}
                  className="text-xs text-[#6B6580] hover:text-brand-600 font-medium flex items-center gap-1 transition-colors"
                  aria-label="从接口查询模型"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                  从接口查询
                </button>
                <button
                  type="button"
                  onClick={addModel}
                  className="text-xs text-brand-600 hover:text-brand-700 font-medium flex items-center gap-1 transition-colors"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  手动添加
                </button>
              </div>
            </div>
            {errors.models && touched.has("models") && (
              <p className="text-xs text-red-500 mb-2" role="alert">{errors.models}</p>
            )}
            <div className="space-y-2">
              {form.models.length === 0 && (
                <div className="text-xs text-[#9C94B0] text-center py-4 border border-dashed border-[#EDE9FE] rounded-xl">
                  暂无模型，点击"添加模型"开始配置
                </div>
              )}
              {form.models.map((entry, i) => (
                <ModelCard
                  key={i}
                  entry={entry}
                  index={i}
                  onChange={handleModelChange}
                  onRemove={removeModel}
                />
              ))}
            </div>
          </div>

          {/* Test connection */}
          <div className="pt-2 border-t border-[#F0EBF5]">
            <div className="flex items-center gap-2 mb-2">
              <select
                value={testModel}
                onChange={(e) => setTestModel(e.target.value)}
                className="input-field flex-1 text-xs"
              >
                <option value="">选择测试模型...</option>
                {form.models
                  .filter((m) => m.name.trim())
                  .map((m) => (
                    <option key={m.name} value={m.name}>
                      {m.name}{m.aliases.length > 0 ? ` (${m.aliases.join(", ")})` : ""}
                    </option>
                  ))}
              </select>
              <button
                type="button"
                onClick={handleTestConnection}
                disabled={testStatus.loading || !testModel.trim()}
                className="flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-[#EDE9FE] text-[#6B6580] hover:bg-white hover:border-brand-600/40 hover:text-brand-600 transition-all disabled:opacity-50 shrink-0"
              >
                {testStatus.loading ? (
                  <>
                    <svg className="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                    测试中
                  </>
                ) : (
                  <>
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                    </svg>
                    测试
                  </>
                )}
              </button>
            </div>
            {testStatus.message && (
              <div
                className={`px-3 py-2 rounded-lg text-xs leading-relaxed ${
                  testStatus.ok
                    ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                    : "bg-red-50 text-red-600 border border-red-200"
                }`}
              >
                {testStatus.message}
              </div>
            )}
          </div>
        </div>
      </Modal>

      {/* Fetch models modal */}
      <Modal
        open={fetchOpen}
        onClose={() => setFetchOpen(false)}
        title="从接口查询模型"
        footer={
          <>
            <button onClick={() => setFetchOpen(false)} className="btn-secondary">取消</button>
            <button
              onClick={addSelectedModels}
              disabled={fetchSelected.size === 0}
              className="btn-primary"
            >
              确认添加 ({fetchSelected.size})
            </button>
          </>
        }
      >
        <div className="space-y-3">
          {fetchLoading ? (
            <div className="flex items-center justify-center py-8">
              <LoadingSpinner />
              <span className="ml-3 text-sm text-[#6B6580]">正在查询 {form.base_url}/v1/models ...</span>
            </div>
          ) : fetchError ? (
            <div className="text-center py-6">
              <p className="text-sm text-red-500 mb-3">{fetchError}</p>
              <button onClick={openFetchModels} className="btn-secondary text-xs">重试</button>
            </div>
          ) : fetchModels.length === 0 ? (
            <div className="text-center py-8">
              <svg className="w-10 h-10 mx-auto text-[#C4BDD5] mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              <p className="text-sm text-[#6B6580]">该接口没有返回可用模型</p>
            </div>
          ) : (
            <>
              {/* Search */}
              <div className="relative">
                <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#9C94B0]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <input
                  type="text"
                  value={fetchSearch}
                  onChange={(e) => setFetchSearch(e.target.value)}
                  placeholder="筛选模型名称..."
                  className="input-field pl-8 text-xs w-full"
                  autoFocus
                />
              </div>
              <p className="text-xs text-[#9C94B0]">{fetchModels.length} 个模型可用{fetchSearch ? `，筛选中` : ""}</p>
              {/* Model list */}
              <div className="max-h-56 overflow-y-auto border border-[#EDE9FE] rounded-xl divide-y divide-[#F0EBF5]">
                {fetchModels
                  .filter((m) => !fetchSearch || m.toLowerCase().includes(fetchSearch.toLowerCase()))
                  .map((name) => {
                    const selected = fetchSelected.has(name);
                    const exists = form.models.some((entry) => entry.name === name);
                    return (
                      <label
                        key={name}
                        className={`flex items-center gap-3 px-3 py-2.5 cursor-pointer transition-colors ${
                          exists ? "opacity-40 cursor-not-allowed bg-[#FAF5FF]" : "hover:bg-brand-50/40"
                        }`}
                      >
                        <input
                          type="checkbox"
                          checked={selected}
                          disabled={exists}
                          onChange={() => toggleFetchModel(name)}
                          className="w-4 h-4 rounded border-[#EDE9FE] text-brand-600 focus:ring-brand-600/20"
                        />
                        <span className="text-sm text-[#1E1B2E] font-mono">{name}</span>
                        {exists && <span className="text-[10px] text-[#9C94B0] ml-auto">已在列表中</span>}
                      </label>
                    );
                  })}
                {fetchModels.filter((m) => !fetchSearch || m.toLowerCase().includes(fetchSearch.toLowerCase())).length ===
                  0 && (
                  <div className="text-center py-6 text-xs text-[#9C94B0]">无匹配的模型</div>
                )}
              </div>
            </>
          )}
          <p className="text-[11px] text-[#9C94B0] leading-relaxed">
            查询 <code className="text-brand-600 bg-brand-50 px-1 rounded">{form.base_url}/v1/models</code>，
            仅添加尚未在下方列表中的模型。勾选后点击「确认添加」。
          </p>
        </div>
      </Modal>

      {/* Delete confirm */}
      <ConfirmDialog
        open={deleteTarget != null}
        title="确认删除 Provider"
        message="删除后将无法恢复，关联的请求日志将标记为「已删除」。"
        danger
        confirmText="确认删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  );
}
