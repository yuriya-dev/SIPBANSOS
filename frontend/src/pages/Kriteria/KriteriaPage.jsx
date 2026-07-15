import { useEffect, useState, useMemo } from "react";
import { toast } from "react-hot-toast";
import { useApi } from "../../hooks/useApi";
import AppShell from "../../components/layout/AppShell";
import Badge from "../../components/ui/Badge";
import Button from "../../components/ui/Button";
import Card from "../../components/ui/Card";
import ProgressBar from "../../components/ui/ProgressBar";
import Drawer from "../../components/ui/Drawer";

const typeVariant = (type) => (type?.toLowerCase() === "benefit" ? "success" : "danger");

const KriteriaPage = () => {
  const {
    getKriteria,
    getKriteriaById,
    updateKriteria,
    createKriteria,
    getKriteriaVersions,
    createKriteriaDefinition,
    updateKriteriaDefinition,
    deleteKriteriaDefinition
  } = useApi();
  
  // Data states
  const [versions, setVersions] = useState([]);
  const [selectedVersionId, setSelectedVersionId] = useState("");
  const [criteria, setCriteria] = useState([]);
  const [versionData, setVersionData] = useState(null);
  
  // Loading states
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  
  // Drawer states
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState("create"); // "create" | "edit"
  const [drawerForm, setDrawerForm] = useState({
    versi: "",
    keterangan: "",
    is_active: false,
    weights: {}
  });

  // Definition Drawer states
  const [isDefDrawerOpen, setIsDefDrawerOpen] = useState(false);
  const [isDefListDrawerOpen, setIsDefListDrawerOpen] = useState(false);
  const [defDrawerMode, setDefDrawerMode] = useState("create"); // "create" | "edit"
  const [defForm, setDefForm] = useState({
    code: "",
    name: "",
    type: "Benefit"
  });

  const fetchVersions = async (selectId = null) => {
    const res = await getKriteriaVersions();
    if (res.success && res.data.length > 0) {
      setVersions(res.data);
      if (selectId) {
        setSelectedVersionId(selectId);
      } else {
        // Default select the active one
        const activeVer = res.data.find(v => v.is_active);
        setSelectedVersionId(activeVer ? activeVer.id : res.data[0].id);
      }
    }
  };

  // Fetch version list on load
  useEffect(() => {
    fetchVersions();
  }, []);

  // Fetch details when selected version changes
  useEffect(() => {
    if (!selectedVersionId) return;

    const fetchDetails = async () => {
      setIsLoading(true);
      const res = await getKriteriaById(selectedVersionId);
      if (res.success) {
        setCriteria(res.data);
        setVersionData(res.version);
      } else {
        toast.error(res.message || "Gagal memuat kriteria.");
      }
      setIsLoading(false);
    };

    fetchDetails();
  }, [selectedVersionId, getKriteriaById]);

  // Derived current weights
  const currentWeights = useMemo(() => {
    const weights = {};
    criteria.forEach((item, idx) => {
      weights[`C${idx + 1}`] = Math.round(item.weight * 100);
    });
    return weights;
  }, [criteria]);

  // Total weight computed on active view (static display)
  const currentTotalWeight = useMemo(() => {
    return Math.round(criteria.reduce((sum, item) => sum + (item.weight || 0), 0) * 100);
  }, [criteria]);

  // Drawer weight sum calculation
  const drawerTotalWeight = useMemo(() => {
    return Object.values(drawerForm.weights).reduce((sum, w) => sum + (Number(w) || 0), 0);
  }, [drawerForm.weights]);

  // Make version active
  const handleActivate = async () => {
    if (!versionData) return;
    setIsSaving(true);
    
    // Prepare weights payload
    const payload = {
      versi: versionData.versi,
      keterangan: versionData.keterangan,
      is_active: true,
      bobot_c1: (Number(currentWeights["C1"]) || 0) / 100,
      bobot_c2: (Number(currentWeights["C2"]) || 0) / 100,
      bobot_c3: (Number(currentWeights["C3"]) || 0) / 100,
      bobot_c4: (Number(currentWeights["C4"]) || 0) / 100,
      bobot_c5: (Number(currentWeights["C5"]) || 0) / 100,
      bobot_c6: (Number(currentWeights["C6"]) || 0) / 100,
      bobot_c7: (Number(currentWeights["C7"]) || 0) / 100,
      bobot_c8: (Number(currentWeights["C8"]) || 0) / 100,
      bobot_c9: (Number(currentWeights["C9"]) || 0) / 100,
      bobot_c10: (Number(currentWeights["C10"]) || 0) / 100,
      bobot_c11: (Number(currentWeights["C11"]) || 0) / 100,
      bobot_c12: (Number(currentWeights["C12"]) || 0) / 100,
      bobot_c13: (Number(currentWeights["C13"]) || 0) / 100,
    };

    const res = await updateKriteria(versionData.id, payload);
    setIsSaving(false);
    if (res.success) {
      toast.success(`Versi ${versionData.versi} berhasil diaktifkan!`);
      fetchVersions(versionData.id);
    } else {
      toast.error(res.message || "Gagal mengaktifkan versi.");
    }
  };

  const openCreateDrawer = () => {
    setDrawerMode("create");
    setDrawerForm({
      versi: "",
      keterangan: "",
      is_active: false,
      weights: { ...currentWeights }
    });
    setIsDrawerOpen(true);
  };

  const openEditDrawer = () => {
    if (!versionData) return;
    setDrawerMode("edit");
    setDrawerForm({
      versi: versionData.versi,
      keterangan: versionData.keterangan || "",
      is_active: versionData.is_active,
      weights: { ...currentWeights }
    });
    setIsDrawerOpen(true);
  };

  const handleDrawerWeightChange = (code, value) => {
    const val = value === "" ? "" : Number(value);
    setDrawerForm(prev => ({
      ...prev,
      weights: {
        ...prev.weights,
        [code]: val
      }
    }));
  };

  const handleDrawerSave = async (e) => {
    e.preventDefault();
    if (!drawerForm.versi.trim()) {
      toast.error("Nama versi wajib diisi.");
      return;
    }
    if (drawerTotalWeight !== 100) {
      toast.error(`Total bobot harus 100%. Saat ini: ${drawerTotalWeight}%`);
      return;
    }

    setIsSaving(true);
    const payload = {
      versi: drawerForm.versi,
      keterangan: drawerForm.keterangan ? drawerForm.keterangan : null,
      is_active: drawerForm.is_active,
      bobot_c1: (Number(drawerForm.weights["C1"]) || 0) / 100,
      bobot_c2: (Number(drawerForm.weights["C2"]) || 0) / 100,
      bobot_c3: (Number(drawerForm.weights["C3"]) || 0) / 100,
      bobot_c4: (Number(drawerForm.weights["C4"]) || 0) / 100,
      bobot_c5: (Number(drawerForm.weights["C5"]) || 0) / 100,
      bobot_c6: (Number(drawerForm.weights["C6"]) || 0) / 100,
      bobot_c7: (Number(drawerForm.weights["C7"]) || 0) / 100,
      bobot_c8: (Number(drawerForm.weights["C8"]) || 0) / 100,
      bobot_c9: (Number(drawerForm.weights["C9"]) || 0) / 100,
      bobot_c10: (Number(drawerForm.weights["C10"]) || 0) / 100,
      bobot_c11: (Number(drawerForm.weights["C11"]) || 0) / 100,
      bobot_c12: (Number(drawerForm.weights["C12"]) || 0) / 100,
      bobot_c13: (Number(drawerForm.weights["C13"]) || 0) / 100,
    };

    let res;
    if (drawerMode === "edit") {
      res = await updateKriteria(versionData.id, payload);
    } else {
      res = await createKriteria(payload);
    }

    setIsSaving(false);
    if (res.success) {
      toast.success(drawerMode === "edit" ? "Kriteria berhasil diperbarui!" : "Versi kriteria baru berhasil dibuat!");
      setIsDrawerOpen(false);
      
      const newVersionId = res.version?.id || selectedVersionId;
      fetchVersions(newVersionId);
    } else {
      toast.error(res.message || "Gagal menyimpan perubahan.");
    }
  };

  const openCreateDefinitionModal = () => {
    setDefDrawerMode("create");
    setDefForm({
      code: "",
      name: "",
      type: "Benefit"
    });
    setIsDefListDrawerOpen(false);
    setIsDefDrawerOpen(true);
  };

  const openEditDefinitionModal = (item) => {
    setDefDrawerMode("edit");
    setDefForm({
      code: item.code,
      name: item.name,
      type: item.type
    });
    setIsDefListDrawerOpen(false);
    setIsDefDrawerOpen(true);
  };

  const handleDefSave = async (e) => {
    e.preventDefault();
    if (!defForm.name.trim()) {
      toast.error("Nama kriteria wajib diisi.");
      return;
    }

    setIsSaving(true);
    let res;
    if (defDrawerMode === "edit") {
      res = await updateKriteriaDefinition(defForm.code, {
        name: defForm.name,
        type: defForm.type
      });
    } else {
      res = await createKriteriaDefinition({
        name: defForm.name,
        type: defForm.type
      });
    }

    setIsSaving(false);
    if (res.success) {
      toast.success(
        defDrawerMode === "edit"
          ? "Kriteria berhasil diperbarui!"
          : "Kriteria baru berhasil ditambahkan!"
      );
      setIsDefDrawerOpen(false);
      setIsDefListDrawerOpen(true);
      if (selectedVersionId) {
        const fetchDetails = async () => {
          setIsLoading(true);
          const detailRes = await getKriteriaById(selectedVersionId);
          if (detailRes.success) {
            setCriteria(detailRes.data);
            setVersionData(detailRes.version);
          }
          setIsLoading(false);
        };
        fetchDetails();
      }
    } else {
      toast.error(res.message || "Gagal menyimpan kriteria.");
    }
  };

  const handleDeleteDefinition = async (item) => {
    if (window.confirm(`Apakah Anda yakin ingin menghapus kriteria ${item.code} (${item.name})?`)) {
      setIsSaving(true);
      const res = await deleteKriteriaDefinition(item.code);
      setIsSaving(false);
      if (res.success) {
        toast.success(`Kriteria ${item.code} berhasil dihapus!`);
        if (selectedVersionId) {
          const fetchDetails = async () => {
            setIsLoading(true);
            const detailRes = await getKriteriaById(selectedVersionId);
            if (detailRes.success) {
              setCriteria(detailRes.data);
              setVersionData(detailRes.version);
            }
            setIsLoading(false);
          };
          fetchDetails();
        }
      } else {
        toast.error(res.message || "Gagal menghapus kriteria.");
      }
    }
  };

  return (
    <AppShell title="Kriteria & Bobot" subtitle="Atur bobot 13 kriteria dan versi per periode." showRightPanel={false}>
      {/* Version Selector Header Card */}
      <Card className="p-4 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-col sm:flex-row sm:items-center gap-3">
          <span className="text-sm font-semibold text-text-secondary">Pilih Versi Kriteria:</span>
          <select
            className="rounded-xl border border-border bg-white px-3 py-2 text-sm text-text-primary outline-none focus:border-primary-orange min-w-[200px]"
            value={selectedVersionId}
            onChange={(e) => setSelectedVersionId(e.target.value)}
            disabled={versions.length === 0}
          >
            {versions.map(v => (
              <option key={v.id} value={v.id}>
                {v.versi} {v.is_active ? "(Aktif)" : ""}
              </option>
            ))}
          </select>
          {versionData && (
            <Badge variant={versionData.is_active ? "success" : "info"}>
              {versionData.is_active ? "Aktif" : "Draft"}
            </Badge>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          {versionData && !versionData.is_active && (
            <Button variant="outline" onClick={handleActivate} disabled={isSaving}>
              {isSaving ? "Memproses..." : "Aktifkan Versi Ini"}
            </Button>
          )}
          <Button variant="outline" onClick={openEditDrawer} disabled={!versionData}>
            Ubah Detail/Bobot
          </Button>
          <Button variant="outline" onClick={() => setIsDefListDrawerOpen(true)} className="border-primary-orange text-primary-orange hover:bg-primary-orange/5">
            Kelola Kriteria
          </Button>
          <Button onClick={openCreateDrawer}>
            Tambah Versi Baru
          </Button>
        </div>
      </Card>

      <div className="grid gap-4 lg:grid-cols-[2.5fr_1fr]">
        {/* Active Version Info & Table */}
        <Card className="p-4">
          <div className="mb-4">
            <h3 className="text-sm font-bold">Daftar Kriteria & Bobot</h3>
            {versionData?.keterangan ? (
              <p className="text-xs text-text-secondary mt-1 italic">
                Keterangan: {versionData.keterangan}
              </p>
            ) : (
              <p className="text-xs text-text-secondary mt-1">
                Keterangan versi ini tidak didefinisikan.
              </p>
            )}
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="text-xs text-text-secondary">
                <tr>
                  <th className="py-2 font-semibold">Kode</th>
                  <th className="py-2 font-semibold">Nama</th>
                  <th className="py-2 font-semibold">Tipe</th>
                  <th className="py-2 font-semibold text-right">Bobot (%)</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/60">
                {isLoading ? (
                  <tr>
                    <td colSpan={4} className="py-6 text-center text-sm text-text-secondary">
                      Memuat kriteria...
                    </td>
                  </tr>
                ) : (
                  criteria.map((item) => (
                    <tr key={item.code}>
                      <td className="py-3 font-semibold text-text-primary">{item.code}</td>
                      <td className="py-3 text-text-secondary">{item.name}</td>
                      <td className="py-3">
                        <Badge variant={typeVariant(item.type)}>{item.type}</Badge>
                      </td>
                      <td className="py-3 font-semibold text-text-primary text-right">
                        {`${(item.weight * 100).toFixed(0)}%`}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Static Sum display */}
        <div className="space-y-4">
          <Card className="p-4">
            <h3 className="text-sm font-bold">Total Bobot Versi Ini</h3>
            <p className="text-xs text-text-secondary">Harus bernilai 100%</p>
            <div className="mt-4 flex items-center justify-between">
              <p className="text-3xl font-bold text-text-primary">{isLoading ? "-" : `${currentTotalWeight}%`}</p>
              {!isLoading && (
                <Badge variant={currentTotalWeight === 100 ? "success" : "warning"}>
                  {currentTotalWeight === 100 ? "Valid" : "Perlu penyesuaian"}
                </Badge>
              )}
            </div>
            {!isLoading && <ProgressBar value={currentTotalWeight} className="mt-3" />}
          </Card>
        </div>
      </div>

      {/* Slide-over Drawer for Create / Edit */}
      <Drawer
        open={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
        title={drawerMode === "create" ? "Tambah Versi Bobot Baru" : "Ubah Versi Bobot"}
        subtitle={
          drawerMode === "create"
            ? "Mulai dengan bobot dasar yang dikloning dari versi saat ini."
            : "Edit kode versi, deskripsi, dan persentase kriteria."
        }
      >
        <form onSubmit={handleDrawerSave} className="flex flex-col gap-4 pb-12">
          {/* Metadata */}
          <div className="space-y-2">
            <label className="text-xs font-semibold text-text-secondary">Nama Versi</label>
            <input
              type="text"
              placeholder="Contoh: v1.1"
              className="w-full rounded-xl border border-border px-3 py-2 text-sm outline-none focus:border-primary-orange"
              value={drawerForm.versi}
              onChange={(e) => setDrawerForm(prev => ({ ...prev, versi: e.target.value }))}
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-semibold text-text-secondary">Deskripsi / Keterangan</label>
            <textarea
              rows={2}
              placeholder="Contoh: Kriteria revisi periode bansos kuartal 3"
              className="w-full rounded-xl border border-border px-3 py-2 text-sm outline-none focus:border-primary-orange"
              value={drawerForm.keterangan}
              onChange={(e) => setDrawerForm(prev => ({ ...prev, keterangan: e.target.value }))}
            />
          </div>
          <div className="flex items-center gap-2 py-2">
            <input
              type="checkbox"
              id="is_active_checkbox"
              className="rounded text-primary-orange focus:ring-primary-orange h-4 w-4"
              checked={drawerForm.is_active}
              onChange={(e) => setDrawerForm(prev => ({ ...prev, is_active: e.target.checked }))}
            />
            <label htmlFor="is_active_checkbox" className="text-xs font-semibold text-text-primary">
              Set versi ini sebagai aktif setelah disimpan
            </label>
          </div>

          <hr className="border-border/60" />

          {/* Sum weight indicator */}
          <div className="bg-background/50 rounded-xl p-3 flex items-center justify-between">
            <div>
              <p className="text-xs font-bold text-text-primary">Total Bobot Form</p>
              <p className="text-[10px] text-text-secondary">Harus tepat 100%</p>
            </div>
            <div className="flex items-center gap-2">
              <span className={`text-xl font-black ${drawerTotalWeight === 100 ? "text-accent-green" : "text-accent-red"}`}>
                {drawerTotalWeight}%
              </span>
              <Badge variant={drawerTotalWeight === 100 ? "success" : "danger"}>
                {drawerTotalWeight === 100 ? "Valid" : "Invalid"}
              </Badge>
            </div>
          </div>

          {/* List of 13 inputs */}
          <div className="space-y-3">
            <label className="text-xs font-semibold text-text-secondary block">Detail Bobot Kriteria</label>
            <div className="space-y-2.5 max-h-[300px] overflow-y-auto pr-1">
              {criteria.map((item) => (
                <div key={item.code} className="flex items-center justify-between gap-3 bg-surface border border-border/40 rounded-xl p-2.5 shadow-sm">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-bold text-text-primary">{item.code}</span>
                      <Badge variant={typeVariant(item.type)} className="text-[9px] px-1 py-0">{item.type}</Badge>
                    </div>
                    <p className="text-xs text-text-secondary truncate mt-0.5">{item.name}</p>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    <input
                      type="number"
                      min="0"
                      max="100"
                      className="w-16 rounded border border-border px-2 py-1 text-sm text-center outline-none focus:border-primary-orange"
                      value={drawerForm.weights[item.code] ?? ""}
                      onChange={(e) => handleDrawerWeightChange(item.code, e.target.value)}
                    />
                    <span className="text-xs text-text-secondary">%</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="mt-4 flex gap-2">
            <Button
              type="submit"
              className="flex-1"
              disabled={isSaving || drawerTotalWeight !== 100 || !drawerForm.versi.trim()}
            >
              {isSaving ? "Menyimpan..." : "Simpan Versi"}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => setIsDrawerOpen(false)}
              disabled={isSaving}
            >
              Batal
            </Button>
          </div>
        </form>
      </Drawer>

      {/* Slide-over Drawer for Criteria Definitions */}
      <Drawer
        open={isDefDrawerOpen}
        onClose={() => {
          setIsDefDrawerOpen(false);
          setIsDefListDrawerOpen(true);
        }}
        title={defDrawerMode === "create" ? "Tambah Kriteria Baru" : "Ubah Detail Kriteria"}
        subtitle={
          defDrawerMode === "create"
            ? "Masukkan nama kriteria baru dan tipenya (Benefit/Cost)."
            : `Mengubah detail untuk kriteria ${defForm.code}`
        }
      >
        <form onSubmit={handleDefSave} className="flex flex-col gap-4 pb-12">
          <div className="space-y-2">
            <label className="text-xs font-semibold text-text-secondary">Nama Kriteria</label>
            <input
              type="text"
              placeholder="Contoh: Pendapatan Bulanan"
              className="w-full rounded-xl border border-border px-3 py-2 text-sm outline-none focus:border-primary-orange"
              value={defForm.name}
              onChange={(e) => setDefForm(prev => ({ ...prev, name: e.target.value }))}
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-semibold text-text-secondary block">Tipe Kriteria</label>
            <select
              className="w-full rounded-xl border border-border px-3 py-2 text-sm outline-none focus:border-primary-orange bg-white"
              value={defForm.type}
              onChange={(e) => setDefForm(prev => ({ ...prev, type: e.target.value }))}
            >
              <option value="Benefit">Benefit (Semakin tinggi semakin baik)</option>
              <option value="Cost">Cost (Semakin rendah semakin baik)</option>
            </select>
          </div>

          <div className="mt-4 flex gap-2">
            <Button
              type="submit"
              className="flex-1"
              disabled={isSaving || !defForm.name.trim()}
            >
              {isSaving ? "Menyimpan..." : "Simpan Kriteria"}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setIsDefDrawerOpen(false);
                setIsDefListDrawerOpen(true);
              }}
              disabled={isSaving}
            >
              Batal
            </Button>
          </div>
        </form>
      </Drawer>

      {/* Slide-over Drawer for Managing Criteria List */}
      <Drawer
        open={isDefListDrawerOpen}
        onClose={() => setIsDefListDrawerOpen(false)}
        title="Kelola Daftar Kriteria"
        subtitle="Kelola kriteria-kriteria penilaian bansos yang aktif."
      >
        <div className="space-y-4 pb-12">
          <div className="flex justify-end">
            <Button
              type="button"
              variant="outline"
              className="border-primary-orange text-primary-orange hover:bg-primary-orange/5 text-xs py-1.5 px-3"
              onClick={() => {
                openCreateDefinitionModal();
              }}
            >
              + Tambah Kriteria Baru
            </Button>
          </div>

          <div className="divide-y divide-border/60">
            {criteria.map((item) => (
              <div key={item.code} className="py-3 flex items-center justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-bold text-sm text-text-primary">{item.code}</span>
                    <Badge variant={typeVariant(item.type)} className="text-[10px] px-1.5 py-0.2">{item.type}</Badge>
                  </div>
                  <p className="text-sm text-text-secondary mt-1 truncate">{item.name}</p>
                </div>
                <div className="flex gap-2 shrink-0">
                  <button
                    type="button"
                    className="text-xs text-primary-orange hover:text-primary-orange-hover font-semibold px-2.5 py-1 border border-border hover:border-primary-orange/30 rounded transition"
                    onClick={() => {
                      openEditDefinitionModal(item);
                    }}
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    className="text-xs text-accent-red hover:text-red-700 font-semibold px-2.5 py-1 border border-border hover:border-accent-red/30 rounded transition"
                    onClick={() => {
                      handleDeleteDefinition(item);
                    }}
                  >
                    Hapus
                  </button>
                </div>
              </div>
            ))}
            {criteria.length === 0 && (
              <p className="text-center text-sm text-text-secondary py-6">Belum ada kriteria.</p>
            )}
          </div>
        </div>
      </Drawer>
    </AppShell>
  );
};

export default KriteriaPage;
