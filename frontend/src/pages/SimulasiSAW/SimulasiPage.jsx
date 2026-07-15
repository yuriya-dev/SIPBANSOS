import { useEffect, useMemo, useState } from "react";
import AppShell from "../../components/layout/AppShell";
import Badge from "../../components/ui/Badge";
import Button from "../../components/ui/Button";
import Card from "../../components/ui/Card";
import { useApi } from "../../hooks/useApi";

const STATUS_VARIANT = {
  Penerima: "penerima",
  Cadangan: "cadangan",
  "Tidak Lolos": "tidak-lolos"
};

const SimulasiPage = () => {
  const { runSAW, getPeriods, getKriteriaVersions } = useApi();
  const [kuota, setKuota] = useState("150");
  const [results, setResults] = useState([]);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState("");
  const [lastRunMs, setLastRunMs] = useState(null);
  const [periods, setPeriods] = useState([]);
  const [selectedPeriodId, setSelectedPeriodId] = useState("");
  const [bobotVersions, setBobotVersions] = useState([]);
  const [selectedBobotId, setSelectedBobotId] = useState("");

  useEffect(() => {
    const fetchPeriodsAndVersions = async () => {
      const pRes = await getPeriods();
      if (pRes.success && pRes.data.length > 0) {
        setPeriods(pRes.data);
        setSelectedPeriodId(pRes.data[0].id);
      }
      const vRes = await getKriteriaVersions();
      if (vRes.success && vRes.data.length > 0) {
        setBobotVersions(vRes.data);
        const activeVer = vRes.data.find(v => v.is_active);
        setSelectedBobotId(activeVer ? activeVer.id : vRes.data[0].id);
      }
    };
    fetchPeriodsAndVersions();
  }, [getPeriods, getKriteriaVersions]);

  const [showAll, setShowAll] = useState(false);

  const displayedResults = useMemo(() => {
    if (showAll) return results;
    return results.slice(0, 10);
  }, [results, showAll]);

  const metrics = useMemo(() => {
    const totalAlternatif = results.length;
    const resolvedKuota = Number.parseInt(kuota, 10) || 0;
    const waktu = lastRunMs ? `${(lastRunMs / 1000).toFixed(2)} detik` : "-";
    return [
      { label: "Total Alternatif", value: totalAlternatif ? String(totalAlternatif) : "-" },
      { label: "Kuota Penerima", value: resolvedKuota ? String(resolvedKuota) : "-" },
      { label: "Waktu Proses", value: waktu }
    ];
  }, [results, kuota, lastRunMs]);

  const handleRun = async () => {
    if (!selectedPeriodId) {
      setError("Silakan pilih periode terlebih dahulu.");
      return;
    }
    setError("");
    setIsRunning(true);
    const start = performance.now();
    const resolvedKuota = Math.max(Number.parseInt(kuota, 10) || 1, 1);
    const result = await runSAW({ 
      kuota: resolvedKuota,
      periodeId: selectedPeriodId,
      bobotId: selectedBobotId
    });
    const elapsed = performance.now() - start;
    setLastRunMs(elapsed);
    setIsRunning(false);

    if (!result.success) {
      setError(result.message || "Gagal menjalankan perhitungan SAW.");
      return;
    }

    setResults(result.data);
    window.dispatchEvent(new Event("calculation-updated"));
  };

  return (
    <AppShell title="Perhitungan SAW" subtitle="Jalankan perhitungan SAW dan analisis hasil ranking." showRightPanel={false}>
      <Card className="p-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div className="grid w-full gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <div>
              <p className="text-xs text-text-secondary">Periode</p>
              <select 
                className="mt-2 w-full rounded-xl border border-border bg-white px-3 py-2 text-sm text-text-primary"
                value={selectedPeriodId}
                onChange={(e) => setSelectedPeriodId(e.target.value)}
              >
                {periods.map(p => (
                  <option key={p.id} value={p.id}>{p.nama_periode}</option>
                ))}
              </select>
            </div>
            <div>
              <p className="text-xs text-text-secondary">Versi Bobot</p>
              <select 
                className="mt-2 w-full rounded-xl border border-border bg-white px-3 py-2 text-sm text-text-primary"
                value={selectedBobotId}
                onChange={(e) => setSelectedBobotId(e.target.value)}
              >
                {bobotVersions.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.versi} {v.is_active ? "(Aktif)" : ""}{v.keterangan ? ` - ${v.keterangan}` : ""}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <p className="text-xs text-text-secondary">Kuota Penerima</p>
              <input
                className="mt-2 w-full rounded-xl border border-border bg-white px-3 py-2 text-sm text-text-primary"
                value={kuota}
                onChange={(event) => setKuota(event.target.value)}
              />
            </div>
          </div>
          <Button onClick={handleRun} disabled={isRunning}>
            {isRunning ? "Memproses..." : "Jalankan Perhitungan"}
          </Button>
        </div>
        {error ? (
          <div className="mt-4 rounded-card bg-accent-red/10 px-3 py-2 text-xs text-accent-red">
            {error}
          </div>
        ) : null}
      </Card>

      <div className="grid gap-4 lg:grid-cols-3">
        {metrics.map((item) => (
          <Card key={item.label} className="p-4">
            <p className="text-sm text-text-secondary">{item.label}</p>
            <p className="mt-2 text-2xl font-bold text-text-primary">{item.value}</p>
          </Card>
        ))}
      </div>

      <Card className="p-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-bold">Hasil Ranking</h3>
            <p className="text-xs text-text-secondary">
              {showAll ? "Menampilkan seluruh hasil perhitungan setelah normalisasi." : "Preview 10 teratas setelah normalisasi."}
            </p>
          </div>
          {results.length > 10 && (
            <Button variant="outline" onClick={() => setShowAll(prev => !prev)}>
              {showAll ? "Tampilkan Lebih Sedikit" : "Lihat Semua"}
            </Button>
          )}
        </div>
        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="text-xs text-text-secondary">
              <tr>
                <th className="py-2 font-semibold">Warga</th>
                <th className="py-2 font-semibold">No. KK</th>
                <th className="py-2 font-semibold">Skor Vi</th>
                <th className="py-2 font-semibold">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {displayedResults.length === 0 ? (
                <tr>
                  <td colSpan={4} className="py-6 text-center text-sm text-text-secondary">
                    Belum ada hasil perhitungan.
                  </td>
                </tr>
              ) : (
                displayedResults.map((item) => (
                  <tr key={`${item.id}-${item.nama}`}>
                    <td className="py-3 font-semibold text-text-primary">{item.nama}</td>
                    <td className="py-3 text-text-secondary">{item.id}</td>
                    <td className="py-3 font-semibold text-text-primary">
                      {Number(item.vi || 0).toFixed(3)}
                    </td>
                    <td className="py-3">
                      <Badge variant={STATUS_VARIANT[item.status] || "info"}>{item.status}</Badge>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </AppShell>
  );
};

export default SimulasiPage;
