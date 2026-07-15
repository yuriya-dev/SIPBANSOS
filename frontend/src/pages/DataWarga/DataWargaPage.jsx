import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import AppShell from "../../components/layout/AppShell";
import Badge from "../../components/ui/Badge";
import Button from "../../components/ui/Button";
import Card from "../../components/ui/Card";
import Drawer from "../../components/ui/Drawer";
import Skeleton from "../../components/ui/Skeleton";
import { useApi } from "../../hooks/useApi";
import { formatNumber, formatRupiah } from "../../utils/formatter";

const PAGE_SIZE = 10;
const STATUS_OPTIONS = ["Semua", "Aktif", "Nonaktif"];
const VERIFICATION_OPTIONS = ["Semua", "Terverifikasi", "Menunggu", "Perlu Revisi"];

const C3_OPTIONS = [
  { value: 1, label: "Skor 1 - Tidak Sekolah / SD" },
  { value: 2, label: "Skor 2 - SMP" },
  { value: 3, label: "Skor 3 - SMA" },
  { value: 4, label: "Skor 4 - Diploma (D1-D4)" },
  { value: 5, label: "Skor 5 - Sarjana (S1-S3)" }
];

const C4_OPTIONS = [
  { value: 1, label: "Skor 1 - Tidak Bekerja / Pengangguran" },
  { value: 2, label: "Skor 2 - Buruh Harian Lepas / Serabutan" },
  { value: 3, label: "Skor 3 - Petani / Nelayan" },
  { value: 4, label: "Skor 4 - Karyawan Swasta / Pedagang" },
  { value: 5, label: "Skor 5 - PNS / TNI / POLRI / BUMN" }
];

const C5_OPTIONS = [
  { value: 1, label: "Skor 1 - Milik Sendiri" },
  { value: 2, label: "Skor 2 - Sewa / Kontrak" },
  { value: 3, label: "Skor 3 - Numpang / Rumah Dinas" }
];

const C7_OPTIONS = [
  { value: 450, label: "450 VA" },
  { value: 900, label: "900 VA" },
  { value: 1300, label: "1300 VA" },
  { value: 2200, label: "2200 VA" },
  { value: 3500, label: "3500 VA" },
  { value: 4400, label: "4400 VA ke atas" }
];

const C12_OPTIONS = [
  { value: 1, label: "Skor 1 - Bambu / Kayu Darurat" },
  { value: 2, label: "Skor 2 - Semi Permanen / Papan" },
  { value: 3, label: "Skor 3 - Tembok Tanpa Plester / Batako" },
  { value: 4, label: "Skor 4 - Tembok Plester / Permanen" }
];

const C13_OPTIONS = [
  { value: 1, label: "Skor 1 - Air Sungai / Hujan / Sumur Terbuka" },
  { value: 2, label: "Skor 2 - Sumur Gali / Pompa Bersama" },
  { value: 3, label: "Skor 3 - PDAM / Sumur Bor Pribadi" }
];

const criteriaFieldsConfig = [
  { field: "c1_value", label: "C1 - Jumlah Anggota Keluarga", type: "number", suffix: "orang" },
  { field: "c2_value", label: "C2 - Jumlah Tanggungan", type: "number", suffix: "orang" },
  { field: "c3_value", label: "C3 - Pendidikan Kep. Keluarga", type: "select", options: C3_OPTIONS },
  { field: "c4_value", label: "C4 - Pekerjaan Kep. Keluarga", type: "select", options: C4_OPTIONS },
  { field: "c5_value", label: "C5 - Status Rumah", type: "select", options: C5_OPTIONS },
  { field: "c6_value", label: "C6 - Luas Rumah", type: "number", suffix: "m²" },
  { field: "c7_value", label: "C7 - Daya Listrik", type: "select", options: C7_OPTIONS },
  { field: "c8_value", label: "C8 - Jumlah Kendaraan", type: "number", suffix: "unit" },
  { field: "c9_value", label: "C9 - Tabungan", type: "number", prefix: "Rp" },
  { field: "c10_value", label: "C10 - Penghasilan per Bulan", type: "number", prefix: "Rp" },
  { field: "c11_value", label: "C11 - Pengeluaran per Bulan", type: "number", prefix: "Rp" },
  { field: "c12_value", label: "C12 - Kondisi Dinding", type: "select", options: C12_OPTIONS },
  { field: "c13_value", label: "C13 - Akses Air", type: "select", options: C13_OPTIONS }
];

const statusVariant = (status) => (status === "Aktif" ? "success" : "danger");

const verificationVariant = (status) => {
  if (status === "Terverifikasi") return "success";
  if (status === "Menunggu") return "warning";
  return "danger";
};

const compactId = (value) => {
  if (!value) return "";
  return value.length <= 12 ? value : `${value.slice(0, 6)}...${value.slice(-4)}`;
};

const formatDate = (value) => {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric"
  }).format(date);
};

const getOriginalFilename = (url) => {
  if (!url) return "";
  const parts = url.split("/");
  const filename = parts[parts.length - 1];
  return filename.replace(/^\d+_\d*_?/, "").replace(/^\d+_?/, "");
};

const buildCriteria = (item, activeCriteria) => {
  const configs = {
    C1: (v) => `${v} orang`,
    C2: (v) => `${v} orang`,
    C3: (v) => `Skor ${v}`,
    C4: (v) => `Skor ${v}`,
    C5: (v) => `Skor ${v}`,
    C6: (v) => `${v} m²`,
    C7: (v) => `${v} VA`,
    C8: (v) => `${v} unit`,
    C9: (v) => formatRupiah(v),
    C10: (v) => formatRupiah(v),
    C11: (v) => formatRupiah(v),
    C12: (v) => `Skor ${v}`,
    C13: (v) => `Skor ${v}`
  };

  const currentCriteria = activeCriteria || [
    { code: "C1", name: "Jumlah Anggota Keluarga" },
    { code: "C2", name: "Jumlah Tanggungan" },
    { code: "C3", name: "Pendidikan Kep. Keluarga" },
    { code: "C4", name: "Pekerjaan Kep. Keluarga" },
    { code: "C5", name: "Status Rumah" },
    { code: "C6", name: "Luas Rumah" },
    { code: "C7", name: "Daya Listrik" },
    { code: "C8", name: "Jumlah Kendaraan" },
    { code: "C9", name: "Tabungan" },
    { code: "C10", name: "Penghasilan per Bulan" },
    { code: "C11", name: "Pengeluaran per Bulan" },
    { code: "C12", name: "Kondisi Dinding" },
    { code: "C13", name: "Akses Air" }
  ];

  return currentCriteria.map(c => {
    const val = item[`${c.code.toLowerCase()}_value`] ?? 0;
    const formatFn = configs[c.code] || ((v) => String(v));
    return {
      label: c.name,
      value: formatFn(val)
    };
  });
};

const resolveVerification = (item) => {
  if (item.foto_ktp_url && item.foto_kk_url) return "Terverifikasi";
  if (item.foto_ktp_url || item.foto_kk_url) return "Perlu Revisi";
  return "Menunggu";
};

const parseRtRwFilter = (value) => {
  if (!value || value === "Semua") return { rt: "", rw: "" };
  const [rt, rw] = value.split("/");
  return { rt: rt?.trim() || "", rw: rw?.trim() || "" };
};

const createEmptyForm = () => ({
  nama_lengkap: "",
  nik: "",
  no_kk: "",
  tanggal_lahir: "",
  jenis_kelamin: "L",
  alamat: "",
  rt: "",
  rw: "",
  no_hp: "",
  foto_ktp_url: "",
  foto_kk_url: "",
  c1_value: "",
  c2_value: "",
  c3_value: "",
  c4_value: "",
  c5_value: "",
  c6_value: "",
  c7_value: "",
  c8_value: "",
  c9_value: "",
  c10_value: "",
  c11_value: "",
  c12_value: "",
  c13_value: ""
});

const toFieldValue = (value) => (value === null || value === undefined ? "" : String(value));

const buildFormDataFromWarga = (item) => ({
  nama_lengkap: item.name || "",
  nik: item.nik || "",
  no_kk: item.noKk || "",
  tanggal_lahir: item.tanggalLahir ? item.tanggalLahir.split("T")[0] : "",
  jenis_kelamin: item.jenisKelaminRaw || "L",
  alamat: item.address || "",
  rt: item.rt || "",
  rw: item.rw || "",
  no_hp: item.no_hp || "",
  foto_ktp_url: item.foto_ktp_url || "",
  foto_kk_url: item.foto_kk_url || "",
  c1_value: toFieldValue(item.c1_value),
  c2_value: toFieldValue(item.c2_value),
  c3_value: toFieldValue(item.c3_value),
  c4_value: toFieldValue(item.c4_value),
  c5_value: toFieldValue(item.c5_value),
  c6_value: toFieldValue(item.c6_value),
  c7_value: toFieldValue(item.c7_value),
  c8_value: toFieldValue(item.c8_value),
  c9_value: toFieldValue(item.c9_value),
  c10_value: toFieldValue(item.c10_value),
  c11_value: toFieldValue(item.c11_value),
  c12_value: toFieldValue(item.c12_value),
  c13_value: toFieldValue(item.c13_value)
});

const buildWargaPayload = (formData, activeCriteria) => {
  const requiredTextFields = ["nama_lengkap", "nik", "no_kk", "tanggal_lahir", "jenis_kelamin", "alamat"];
  for (const field of requiredTextFields) {
    if (!String(formData[field] || "").trim()) {
      throw new Error("Lengkapi semua data wajib sebelum menyimpan.");
    }
  }

  const payload = {
    nama_lengkap: String(formData.nama_lengkap || "").trim(),
    nik: String(formData.nik || "").trim(),
    no_kk: String(formData.no_kk || "").trim(),
    tanggal_lahir: String(formData.tanggal_lahir || "").trim(),
    jenis_kelamin: String(formData.jenis_kelamin || "L").trim(),
    alamat: String(formData.alamat || "").trim(),
    rt: String(formData.rt || "").trim(),
    rw: String(formData.rw || "").trim(),
    no_hp: String(formData.no_hp || "").trim(),
    foto_ktp_url: String(formData.foto_ktp_url || "").trim(),
    foto_kk_url: String(formData.foto_kk_url || "").trim()
  };

  const numericFields = [
    "c1_value",
    "c2_value",
    "c3_value",
    "c4_value",
    "c5_value",
    "c6_value",
    "c7_value",
    "c8_value",
    "c9_value",
    "c10_value",
    "c11_value",
    "c12_value",
    "c13_value"
  ];

  for (const field of numericFields) {
    const code = field.split("_")[0].toUpperCase();
    const isActive = activeCriteria ? activeCriteria.some(c => c.code === code) : true;
    let value = 0;
    if (isActive) {
      value = Number(formData[field]);
      if (Number.isNaN(value) || formData[field] === "") {
        const cName = activeCriteria ? (activeCriteria.find(c => c.code === code)?.name || "") : "";
        throw new Error(`Nilai ${code} (${cName || field.replace("_", " ")}) harus diisi dengan angka.`);
      }
    }
    payload[field] = value;
  }

  return payload;
};

const mapWargaResponse = (item, activeCriteria) => {
  const rt = item.rt ?? "";
  const rw = item.rw ?? "";
  const rtRw = rt && rw ? `${rt}/${rw}` : rt || rw || "-";
  const documents = {
    ktp: item.foto_ktp_url ? "Lengkap" : "Kurang",
    kk: item.foto_kk_url ? "Lengkap" : "Kurang"
  };
  const verification = resolveVerification(item);

  return {
    id: item.id,
    name: item.nama_lengkap,
    nik: item.nik,
    noKk: item.no_kk,
    tanggalLahir: item.tanggal_lahir,
    jenisKelaminRaw: item.jenis_kelamin,
    rtRw,
    rt,
    rw,
    address: item.alamat,
    gender: item.jenis_kelamin === "L" ? "Laki-laki" : "Perempuan",
    birthDate: formatDate(item.tanggal_lahir),
    phone: item.no_hp || "-",
    no_hp: item.no_hp || "",
    foto_ktp_url: item.foto_ktp_url || "",
    foto_kk_url: item.foto_kk_url || "",
    c1_value: item.c1_value,
    c2_value: item.c2_value,
    c3_value: item.c3_value,
    c4_value: item.c4_value,
    c5_value: item.c5_value,
    c6_value: item.c6_value,
    c7_value: item.c7_value,
    c8_value: item.c8_value,
    c9_value: item.c9_value,
    c10_value: item.c10_value,
    c11_value: item.c11_value,
    c12_value: item.c12_value,
    c13_value: item.c13_value,
    penghasilan: item.c10_value,
    tanggungan: item.c2_value,
    status: item.is_active ? "Aktif" : "Nonaktif",
    verification,
    updated: formatDate(item.updated_at),
    documents,
    criteria: buildCriteria(item, activeCriteria),
    source: item
  };
};

const defaultCriteria = [
  { code: "C1", name: "Jumlah Anggota Keluarga", type: "Benefit" },
  { code: "C2", name: "Jumlah Tanggungan", type: "Benefit" },
  { code: "C3", name: "Pendidikan Kep. Keluarga", type: "Cost" },
  { code: "C4", name: "Pekerjaan Kep. Keluarga", type: "Cost" },
  { code: "C5", name: "Status Rumah", type: "Benefit" },
  { code: "C6", name: "Luas Rumah", type: "Cost" },
  { code: "C7", name: "Daya Listrik", type: "Cost" },
  { code: "C8", name: "Jumlah Kendaraan", type: "Cost" },
  { code: "C9", name: "Tabungan", type: "Cost" },
  { code: "C10", name: "Penghasilan per Bulan", type: "Cost" },
  { code: "C11", name: "Pengeluaran per Bulan", type: "Benefit" },
  { code: "C12", name: "Kondisi Dinding", type: "Cost" },
  { code: "C13", name: "Akses Air", type: "Cost" }
];

const DataWargaPage = () => {
  const { getKriteria, getWarga, createWarga, updateWarga, uploadFile, getWargaHistory } = useApi();
  const navigate = useNavigate();
  const [activeCriteria, setActiveCriteria] = useState(defaultCriteria);
  const [query, setQuery] = useState("");
  const [selectedStatus, setSelectedStatus] = useState("Semua");
  const [selectedRtRw, setSelectedRtRw] = useState("Semua");
  const [selectedVerification, setSelectedVerification] = useState("Semua");
  const [selectedWarga, setSelectedWarga] = useState(null);
  const [warga, setWarga] = useState([]);
  const [page, setPage] = useState(1);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [total, setTotal] = useState(0);

  useEffect(() => {
    const fetchCriteria = async () => {
      const res = await getKriteria();
      if (res.success && res.data.length > 0) {
        setActiveCriteria(res.data.map(item => ({
          code: item.code,
          name: item.name,
          type: item.type
        })));
      }
    };
    fetchCriteria();
  }, [getKriteria]);

  const dynamicCriteriaFieldsConfig = useMemo(() => {
    const configs = {
      C1: { type: "number", suffix: "orang" },
      C2: { type: "number", suffix: "orang" },
      C3: { type: "select", options: C3_OPTIONS },
      C4: { type: "select", options: C4_OPTIONS },
      C5: { type: "select", options: C5_OPTIONS },
      C6: { type: "number", suffix: "m²" },
      C7: { type: "select", options: C7_OPTIONS },
      C8: { type: "number", suffix: "unit" },
      C9: { type: "number", prefix: "Rp" },
      C10: { type: "number", prefix: "Rp" },
      C11: { type: "number", prefix: "Rp" },
      C12: { type: "select", options: C12_OPTIONS },
      C13: { type: "select", options: C13_OPTIONS }
    };

    return activeCriteria.map(item => {
      const code = item.code;
      const conf = configs[code] || { type: "number" };
      return {
        field: `${code.toLowerCase()}_value`,
        label: `${code} - ${item.name}`,
        ...conf
      };
    });
  }, [activeCriteria]);
  const [statsData, setStatsData] = useState({ active_count: 0, pending_count: 0, missing_docs_count: 0 });

  const rtRwOptions = useMemo(() => {
    const unique = new Set(
      warga
        .map((item) => item.rtRw)
        .filter((value) => value && value !== "-")
    );
    return ["Semua", ...Array.from(unique).sort()];
  }, [warga]);

  useEffect(() => {
    if (!rtRwOptions.includes(selectedRtRw)) {
      setSelectedRtRw("Semua");
    }
  }, [rtRwOptions, selectedRtRw]);

  useEffect(() => {
    setPage(1);
  }, [query, selectedRtRw]);

  useEffect(() => {
    let isActive = true;
    const fetchData = async () => {
      setIsLoading(true);
      setError("");
      const { rt, rw } = parseRtRwFilter(selectedRtRw);
      const result = await getWarga({
        page,
        limit: PAGE_SIZE,
        q: query.trim(),
        rt,
        rw
      });

      if (!isActive) return;

      if (!result.success) {
        setError(result.message || "Gagal memuat data warga.");
        setWarga([]);
        setTotal(0);
        setStatsData({ active_count: 0, pending_count: 0, missing_docs_count: 0 });
        setIsLoading(false);
        return;
      }

      setWarga(result.data.map(item => mapWargaResponse(item, activeCriteria)));
      setTotal(result.total || 0);
      setStatsData(result.stats || { active_count: 0, pending_count: 0, missing_docs_count: 0 });
      setIsLoading(false);
    };

    fetchData();
    return () => {
      isActive = false;
    };
  }, [getWarga, page, query, selectedRtRw, activeCriteria]);

  const filteredWarga = useMemo(() => {
    return warga.filter((item) => {
      const matchesStatus = selectedStatus === "Semua" || item.status === selectedStatus;
      const matchesVerification =
        selectedVerification === "Semua" || item.verification === selectedVerification;
      return matchesStatus && matchesVerification;
    });
  }, [warga, selectedStatus, selectedVerification]);

  const stats = useMemo(() => {
    const activeCount = statsData.active_count;
    const pendingCount = statsData.pending_count;
    const missingDocs = statsData.missing_docs_count;

    return [
      {
        label: "Warga Aktif",
        value: formatNumber(activeCount),
        helper: "Daftar saat ini",
        variant: "success",
        note: "Data aktif siap dihitung."
      },
      {
        label: "Menunggu Verifikasi",
        value: formatNumber(pendingCount),
        helper: "Perlu tindak lanjut",
        variant: "info",
        note: "Dokumen belum lengkap."
      },
      {
        label: "Dokumen Kurang",
        value: formatNumber(missingDocs),
        helper: "Butuh revisi",
        variant: "warning",
        note: "KTP atau KK belum sesuai."
      }
    ];
  }, [statsData]);

  const hasNextPage = page * PAGE_SIZE < total;

  const closeDrawer = () => {
    setSelectedWarga(null);
    setShowForm(false);
    setShowHistory(false);
  };

  const openCreateForm = () => {
    setIsEditing(false);
    setFormError("");
    setFormData(createEmptyForm());
    setShowForm(true);
  };

  const openEditForm = () => {
    if (!selectedWarga) return;
    setIsEditing(true);
    setFormError("");
    setFormData(buildFormDataFromWarga(selectedWarga));
    setShowForm(true);
  };

  const closeForm = () => setShowForm(false);
  const closeHistory = () => setShowHistory(false);

  const [showForm, setShowForm] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [formData, setFormData] = useState(createEmptyForm());
  const [formLoading, setFormLoading] = useState(false);
  const [formError, setFormError] = useState("");

  const [isUploadingKtp, setIsUploadingKtp] = useState(false);
  const [uploadKtpError, setUploadKtpError] = useState("");

  const handleKtpUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setIsUploadingKtp(true);
    setUploadKtpError("");
    try {
      const result = await uploadFile(file);
      if (result && !result.success) {
        throw new Error(result.message);
      }
      setFormData((prev) => ({ ...prev, foto_ktp_url: result.url }));
    } catch (err) {
      setUploadKtpError(err.message || "Gagal mengunggah foto KTP.");
    } finally {
      setIsUploadingKtp(false);
    }
  };

  const [isUploadingKk, setIsUploadingKk] = useState(false);
  const [uploadKkError, setUploadKkError] = useState("");
  const [previewImage, setPreviewImage] = useState(null);

  const handleKkUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setIsUploadingKk(true);
    setUploadKkError("");
    try {
      const result = await uploadFile(file);
      if (result && !result.success) {
        throw new Error(result.message);
      }
      setFormData((prev) => ({ ...prev, foto_kk_url: result.url }));
    } catch (err) {
      setUploadKkError(err.message || "Gagal mengunggah foto KK.");
    } finally {
      setIsUploadingKk(false);
    }
  };

  const [showHistory, setShowHistory] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyError, setHistoryError] = useState("");
  const [historyRecords, setHistoryRecords] = useState([]);

  useEffect(() => {
    if (!showHistory || !selectedWarga) return;

    let isActive = true;
    const fetchHistory = async () => {
      setHistoryLoading(true);
      setHistoryError("");
      const result = await getWargaHistory(selectedWarga.id);

      if (!isActive) return;

      if (!result.success) {
        setHistoryError(result.message || "Gagal memuat riwayat.");
        setHistoryRecords([]);
        setHistoryLoading(false);
        return;
      }

      setHistoryRecords(result.data || []);
      setHistoryLoading(false);
    };

    fetchHistory();
    return () => {
      isActive = false;
    };
  }, [getWargaHistory, selectedWarga, showHistory]);

  const refreshWargaList = async () => {
    setIsLoading(true);
    const { rt, rw } = parseRtRwFilter(selectedRtRw);
    const result = await getWarga({ page, limit: PAGE_SIZE, q: query.trim(), rt, rw });
    if (!result.success) {
      setError(result.message || "Gagal memuat data warga.");
      setWarga([]);
      setTotal(0);
      setStatsData({ active_count: 0, pending_count: 0, missing_docs_count: 0 });
      setIsLoading(false);
      return;
    }
    setWarga(result.data.map(item => mapWargaResponse(item, activeCriteria)));
    setTotal(result.total || 0);
    setStatsData(result.stats || { active_count: 0, pending_count: 0, missing_docs_count: 0 });
    setIsLoading(false);
  };

  const handleSubmitForm = async () => {
    setFormError("");
    setFormLoading(true);
    try {
      const payload = buildWargaPayload(formData, activeCriteria);

      let result;
      if (isEditing && selectedWarga) {
        result = await updateWarga(selectedWarga.id, payload);
      } else {
        result = await createWarga(payload);
      }

      if (result && !result.success) {
        throw new Error(result.message);
      }

      setShowForm(false);
      await refreshWargaList();
      if (isEditing && selectedWarga && result && result.data) {
        setSelectedWarga(mapWargaResponse(result.data, activeCriteria));
      }
    } catch (err) {
      setFormError(err.message || "Gagal menyimpan data.");
    } finally {
      setFormLoading(false);
    }
  };



  return (
    <AppShell
      title="Data Warga"
      subtitle="Kelola data warga berdasarkan 13 kriteria dan status penyaluran."
      showRightPanel={false}
    >
      <div className="grid gap-4 lg:grid-cols-3">
        {stats.map((item) => (
          <Card key={item.label} className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-text-secondary">{item.label}</p>
                <p className="text-2xl font-bold text-text-primary">{item.value}</p>
              </div>
              <Badge variant={item.variant}>{item.helper}</Badge>
            </div>
            <p className="mt-4 text-xs text-text-secondary">{item.note}</p>
          </Card>
        ))}
      </div>

      <Card className="p-4">
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div className="flex-1 min-w-0">
              <input
                className="w-full rounded-button border border-border bg-white px-4 py-2 text-sm text-text-primary outline-none focus:border-primary-orange"
                placeholder="Cari nama, NIK, atau nomor KK"
                aria-label="Cari data warga"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
              />
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                onClick={() => navigate("/import-export?tab=import")}
              >
                Import
              </Button>
              <Button
                variant="outline"
                onClick={() => navigate("/import-export?tab=export")}
              >
                Ekspor
              </Button>
              <Button onClick={openCreateForm}>Tambah Warga</Button>
            </div>
          </div>
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <div className="flex flex-col gap-1">
              <label className="text-xs font-semibold text-text-secondary">RT/RW</label>
              <select
                className="rounded-button border border-border bg-white px-4 py-2 text-sm text-text-primary"
                value={selectedRtRw}
                onChange={(event) => setSelectedRtRw(event.target.value)}
              >
                {rtRwOptions.map((option) => (
                  <option key={option} value={option}>
                    {option === "Semua" ? "Semua RT/RW" : `RT/RW ${option}`}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs font-semibold text-text-secondary">Status</label>
              <select
                className="rounded-button border border-border bg-white px-4 py-2 text-sm text-text-primary"
                value={selectedStatus}
                onChange={(event) => setSelectedStatus(event.target.value)}
              >
                {STATUS_OPTIONS.map((option) => (
                  <option key={option} value={option}>
                    {option === "Semua" ? "Semua Status" : option}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs font-semibold text-text-secondary">Verifikasi</label>
              <select
                className="rounded-button border border-border bg-white px-4 py-2 text-sm text-text-primary"
                value={selectedVerification}
                onChange={(event) => setSelectedVerification(event.target.value)}
              >
                {VERIFICATION_OPTIONS.map((option) => (
                  <option key={option} value={option}>
                    {option === "Semua" ? "Semua Verifikasi" : option}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {error ? (
          <div className="mt-4 rounded-card bg-accent-red/10 px-3 py-2 text-xs text-accent-red">
            {error}
          </div>
        ) : null}

        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="text-xs text-text-secondary">
              <tr>
                <th className="py-2 font-semibold">Nama Warga</th>
                <th className="py-2 font-semibold w-[140px]">NIK</th>
                <th className="py-2 font-semibold w-[140px]">No. KK</th>
                <th className="py-2 font-semibold">RT/RW</th>
                <th className="py-2 font-semibold">Penghasilan</th>
                <th className="py-2 font-semibold">Tanggungan</th>
                <th className="py-2 font-semibold">Status</th>
                <th className="py-2 font-semibold">Verifikasi</th>
                <th className="py-2 font-semibold">Terakhir Update</th>
                <th className="py-2 font-semibold">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {isLoading ? (
                [...Array(5)].map((_, i) => (
                  <tr key={`skeleton-${i}`} className="hover:bg-transparent">
                    <td className="py-3"><Skeleton className="h-5 w-32" /></td>
                    <td className="py-3"><Skeleton className="h-5 w-32" /></td>
                    <td className="py-3"><Skeleton className="h-5 w-32" /></td>
                    <td className="py-3"><Skeleton className="h-5 w-16" /></td>
                    <td className="py-3"><Skeleton className="h-5 w-24" /></td>
                    <td className="py-3"><Skeleton className="h-5 w-16" /></td>
                    <td className="py-3"><Skeleton className="h-6 w-20 rounded-full" /></td>
                    <td className="py-3"><Skeleton className="h-6 w-24 rounded-full" /></td>
                    <td className="py-3"><Skeleton className="h-5 w-24" /></td>
                    <td className="py-3"><Skeleton className="h-8 w-16 rounded-button" /></td>
                  </tr>
                ))
              ) : filteredWarga.length === 0 ? (
                <tr>
                  <td colSpan={10} className="py-6 text-center text-sm text-text-secondary">
                    Belum ada data warga yang cocok.
                  </td>
                </tr>
              ) : (
                filteredWarga.map((item) => (
                  <tr key={item.nik} className="hover:bg-background/60">
                    <td className="py-3 font-semibold text-text-primary">{item.name}</td>
                    <td className="py-3 text-text-secondary font-mono tabular-nums" title={item.nik}>
                      {compactId(item.nik)}
                    </td>
                    <td className="py-3 text-text-secondary font-mono tabular-nums" title={item.noKk}>
                      {compactId(item.noKk)}
                    </td>
                    <td className="py-3 text-text-secondary">{item.rtRw}</td>
                    <td className="py-3 text-text-secondary">{formatRupiah(item.penghasilan)}</td>
                    <td className="py-3 text-text-secondary">{item.tanggungan} orang</td>
                    <td className="py-3">
                      <Badge variant={statusVariant(item.status)}>{item.status}</Badge>
                    </td>
                    <td className="py-3">
                      <Badge variant={verificationVariant(item.verification)}>{item.verification}</Badge>
                    </td>
                    <td className="py-3 text-text-secondary">{item.updated}</td>
                    <td className="py-3">
                      <Button variant="outline" onClick={() => setSelectedWarga(item)}>
                        Detail
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        <div className="mt-4 flex flex-wrap items-center justify-between text-xs text-text-secondary">
          <span>
            Menampilkan {filteredWarga.length} data pada halaman ini • Total {formatNumber(total)} data
          </span>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              onClick={() => setPage((prev) => Math.max(prev - 1, 1))}
              disabled={page === 1}
            >
              Sebelumnya
            </Button>
            <Button
              variant="ghost"
              onClick={() => setPage((prev) => prev + 1)}
              disabled={!hasNextPage}
            >
              Berikutnya
            </Button>
          </div>
        </div>
      </Card>

      <Drawer
        open={Boolean(selectedWarga)}
        onClose={closeDrawer}
        title="Detail Warga"
        subtitle={selectedWarga ? `${selectedWarga.name} • ${selectedWarga.nik}` : ""}
      >
        {selectedWarga ? (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={statusVariant(selectedWarga.status)}>{selectedWarga.status}</Badge>
              <Badge variant={verificationVariant(selectedWarga.verification)}>
                {selectedWarga.verification}
              </Badge>
              <span className="text-xs text-text-secondary">Update {selectedWarga.updated}</span>
            </div>

            <div>
              <h3 className="text-sm font-bold">Identitas</h3>
              <div className="mt-3 grid gap-3 sm:grid-cols-2">
                <div className="rounded-card bg-background/70 p-3">
                  <p className="text-xs text-text-secondary">NIK</p>
                  <p className="mt-1 text-sm font-semibold text-text-primary font-mono tabular-nums">
                    {selectedWarga.nik}
                  </p>
                </div>
                <div className="rounded-card bg-background/70 p-3">
                  <p className="text-xs text-text-secondary">No. KK</p>
                  <p className="mt-1 text-sm font-semibold text-text-primary font-mono tabular-nums">
                    {selectedWarga.noKk}
                  </p>
                </div>
                <div className="rounded-card bg-background/70 p-3">
                  <p className="text-xs text-text-secondary">Alamat</p>
                  <p className="mt-1 text-sm font-semibold text-text-primary">{selectedWarga.address}</p>
                </div>
                <div className="rounded-card bg-background/70 p-3">
                  <p className="text-xs text-text-secondary">RT/RW</p>
                  <p className="mt-1 text-sm font-semibold text-text-primary">{selectedWarga.rtRw}</p>
                </div>
                <div className="rounded-card bg-background/70 p-3">
                  <p className="text-xs text-text-secondary">Jenis Kelamin</p>
                  <p className="mt-1 text-sm font-semibold text-text-primary">{selectedWarga.gender}</p>
                </div>
                <div className="rounded-card bg-background/70 p-3">
                  <p className="text-xs text-text-secondary">Tanggal Lahir</p>
                  <p className="mt-1 text-sm font-semibold text-text-primary">{selectedWarga.birthDate}</p>
                </div>
                <div className="rounded-card bg-background/70 p-3">
                  <p className="text-xs text-text-secondary">No. HP</p>
                  <p className="mt-1 text-sm font-semibold text-text-primary">{selectedWarga.phone}</p>
                </div>
                <div className="rounded-card bg-background/70 p-3">
                  <p className="text-xs text-text-secondary">Penghasilan</p>
                  <p className="mt-1 text-sm font-semibold text-text-primary">
                    {formatRupiah(selectedWarga.penghasilan)}
                  </p>
                </div>
              </div>
            </div>

            <div>
              <h3 className="text-sm font-bold">Kriteria SAW</h3>
              <div className="mt-3 grid gap-3 sm:grid-cols-2">
                {selectedWarga.criteria.map((item) => (
                  <div key={item.label} className="rounded-card bg-background/70 p-3">
                    <p className="text-xs text-text-secondary">{item.label}</p>
                    <p className="mt-1 text-sm font-semibold text-text-primary">{item.value}</p>
                  </div>
                ))}
              </div>
            </div>

            <div>
              <h3 className="text-sm font-bold">Dokumen (Klik untuk Lihat)</h3>
              <div className="mt-3 grid gap-3 sm:grid-cols-2">
                <div
                  className={`rounded-card bg-background/70 p-3 transition-all ${
                    selectedWarga.foto_ktp_url
                      ? "cursor-pointer hover:bg-background/95 hover:shadow-sm border border-transparent hover:border-border/80"
                      : ""
                  }`}
                  onClick={() => {
                    if (selectedWarga.foto_ktp_url) {
                      setPreviewImage(selectedWarga.foto_ktp_url);
                    }
                  }}
                >
                  <p className="text-xs text-text-secondary flex justify-between items-center">
                    <span>KTP</span>
                    {selectedWarga.foto_ktp_url ? (
                      <span className="text-[10px] text-primary-orange font-semibold hover:underline">Lihat</span>
                    ) : null}
                  </p>
                  <div className="mt-1">
                    <Badge variant={selectedWarga.documents.ktp === "Lengkap" ? "success" : "warning"}>
                      {selectedWarga.documents.ktp}
                    </Badge>
                  </div>
                </div>
                <div
                  className={`rounded-card bg-background/70 p-3 transition-all ${
                    selectedWarga.foto_kk_url
                      ? "cursor-pointer hover:bg-background/95 hover:shadow-sm border border-transparent hover:border-border/80"
                      : ""
                  }`}
                  onClick={() => {
                    if (selectedWarga.foto_kk_url) {
                      setPreviewImage(selectedWarga.foto_kk_url);
                    }
                  }}
                >
                  <p className="text-xs text-text-secondary flex justify-between items-center">
                    <span>KK</span>
                    {selectedWarga.foto_kk_url ? (
                      <span className="text-[10px] text-primary-orange font-semibold hover:underline">Lihat</span>
                    ) : null}
                  </p>
                  <div className="mt-1">
                    <Badge variant={selectedWarga.documents.kk === "Lengkap" ? "success" : "warning"}>
                      {selectedWarga.documents.kk}
                    </Badge>
                  </div>
                </div>
              </div>
            </div>

            <div className="flex flex-wrap gap-2">
              <Button onClick={openEditForm}>Ubah Data</Button>
              <Button variant="outline" onClick={() => setShowHistory(true)}>Lihat Riwayat</Button>
            </div>
          </div>
        ) : null}
      </Drawer>
      <Drawer
        open={showForm}
        onClose={closeForm}
        title={isEditing ? "Ubah Warga" : "Tambah Warga"}
      >
        <div className="space-y-4">
          {formError ? <div className="rounded-card bg-accent-red/10 px-3 py-2 text-xs text-accent-red">{formError}</div> : null}
          <div className="grid gap-3 sm:grid-cols-2">
            <input className="rounded-button border border-border px-3 py-2 sm:col-span-2" placeholder="Nama Lengkap" value={formData.nama_lengkap} onChange={(e) => setFormData({ ...formData, nama_lengkap: e.target.value })} />
            <input className="rounded-button border border-border px-3 py-2" placeholder="NIK" value={formData.nik} onChange={(e) => setFormData({ ...formData, nik: e.target.value })} />
            <input className="rounded-button border border-border px-3 py-2" placeholder="No. KK" value={formData.no_kk} onChange={(e) => setFormData({ ...formData, no_kk: e.target.value })} />
            <input className="rounded-button border border-border px-3 py-2 sm:col-span-2" type="date" placeholder="Tanggal Lahir" value={formData.tanggal_lahir} onChange={(e) => setFormData({ ...formData, tanggal_lahir: e.target.value })} />
            <select className="rounded-button border border-border px-3 py-2" value={formData.jenis_kelamin} onChange={(e) => setFormData({ ...formData, jenis_kelamin: e.target.value })}>
              <option value="L">Laki-laki</option>
              <option value="P">Perempuan</option>
            </select>
            <input className="rounded-button border border-border px-3 py-2" placeholder="No. HP" value={formData.no_hp} onChange={(e) => setFormData({ ...formData, no_hp: e.target.value })} />
            <input className="rounded-button border border-border px-3 py-2 sm:col-span-2" placeholder="Alamat" value={formData.alamat} onChange={(e) => setFormData({ ...formData, alamat: e.target.value })} />
            <input className="rounded-button border border-border px-3 py-2" placeholder="RT" value={formData.rt} onChange={(e) => setFormData({ ...formData, rt: e.target.value })} />
            <input className="rounded-button border border-border px-3 py-2" placeholder="RW" value={formData.rw} onChange={(e) => setFormData({ ...formData, rw: e.target.value })} />
            <div className="sm:col-span-2 flex flex-col gap-1">
              <label className="text-xs font-semibold text-text-secondary">Foto KTP</label>
              {formData.foto_ktp_url ? (
                <div className="flex flex-col gap-2">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-emerald-600 font-semibold flex items-center gap-1">
                      <span className="inline-block w-2 h-2 rounded-full bg-emerald-500"></span>
                      Foto KTP terunggah
                    </span>
                    <button
                      type="button"
                      className="text-xs text-accent-red underline hover:text-accent-red/80 font-medium"
                      onClick={() => setFormData({ ...formData, foto_ktp_url: "" })}
                    >
                      Hapus & Upload Ulang
                    </button>
                  </div>
                  <input
                    className="rounded-button border border-border px-3 py-2 bg-background/40 text-sm text-text-secondary w-full select-all font-mono text-xs cursor-not-allowed"
                    value={getOriginalFilename(formData.foto_ktp_url)}
                    disabled
                    readOnly
                  />
                </div>
              ) : (
                <div className="w-full">
                  <label
                    htmlFor="upload-ktp"
                    className="flex flex-col items-center justify-center border-2 border-dashed border-border hover:border-primary-orange rounded-card p-4 cursor-pointer text-sm text-text-secondary transition-colors bg-white hover:bg-background/20"
                  >
                    {isUploadingKtp ? (
                      <span className="text-primary-orange animate-pulse font-medium">Mengunggah...</span>
                    ) : (
                      <div className="flex flex-col items-center gap-1">
                        <span className="font-semibold text-primary-orange">Klik untuk Upload</span>
                        <span className="text-xs text-text-secondary/70">JPG, PNG, WEBP (Maks 5MB)</span>
                      </div>
                    )}
                  </label>
                  <input
                    id="upload-ktp"
                    type="file"
                    accept="image/*"
                    className="hidden"
                    disabled={isUploadingKtp}
                    onChange={handleKtpUpload}
                  />
                </div>
              )}
              {uploadKtpError ? (
                <span className="text-xs text-accent-red font-medium">{uploadKtpError}</span>
              ) : null}
            </div>
            <div className="sm:col-span-2 flex flex-col gap-1">
              <label className="text-xs font-semibold text-text-secondary">Foto KK</label>
              {formData.foto_kk_url ? (
                <div className="flex flex-col gap-2">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-emerald-600 font-semibold flex items-center gap-1">
                      <span className="inline-block w-2 h-2 rounded-full bg-emerald-500"></span>
                      Foto KK terunggah
                    </span>
                    <button
                      type="button"
                      className="text-xs text-accent-red underline hover:text-accent-red/80 font-medium"
                      onClick={() => setFormData({ ...formData, foto_kk_url: "" })}
                    >
                      Hapus & Upload Ulang
                    </button>
                  </div>
                  <input
                    className="rounded-button border border-border px-3 py-2 bg-background/40 text-sm text-text-secondary w-full select-all font-mono text-xs cursor-not-allowed"
                    value={getOriginalFilename(formData.foto_kk_url)}
                    disabled
                    readOnly
                  />
                </div>
              ) : (
                <div className="w-full">
                  <label
                    htmlFor="upload-kk"
                    className="flex flex-col items-center justify-center border-2 border-dashed border-border hover:border-primary-orange rounded-card p-4 cursor-pointer text-sm text-text-secondary transition-colors bg-white hover:bg-background/20"
                  >
                    {isUploadingKk ? (
                      <span className="text-primary-orange animate-pulse font-medium">Mengunggah...</span>
                    ) : (
                      <div className="flex flex-col items-center gap-1">
                        <span className="font-semibold text-primary-orange">Klik untuk Upload</span>
                        <span className="text-xs text-text-secondary/70">JPG, PNG, WEBP (Maks 5MB)</span>
                      </div>
                    )}
                  </label>
                  <input
                    id="upload-kk"
                    type="file"
                    accept="image/*"
                    className="hidden"
                    disabled={isUploadingKk}
                    onChange={handleKkUpload}
                  />
                </div>
              )}
              {uploadKkError ? (
                <span className="text-xs text-accent-red font-medium">{uploadKkError}</span>
              ) : null}
            </div>
          </div>

          <div className="border-t border-border/60 my-4 pt-4">
            <h4 className="text-sm font-bold text-text-primary mb-3">Kriteria Penilaian SAW</h4>
            <div className="grid gap-4 sm:grid-cols-2">
              {dynamicCriteriaFieldsConfig.map((cfg) => {
                if (cfg.type === "select") {
                  return (
                    <div key={cfg.field} className="flex flex-col gap-1">
                      <label className="text-xs font-semibold text-text-secondary">{cfg.label}</label>
                      <select
                        className="rounded-button border border-border px-3 py-2 text-sm bg-white text-text-primary"
                        value={formData[cfg.field]}
                        onChange={(e) => setFormData({ ...formData, [cfg.field]: e.target.value })}
                      >
                        <option value="">-- Pilih {cfg.label.split(" - ")[1]} --</option>
                        {cfg.options.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {opt.label}
                          </option>
                        ))}
                      </select>
                    </div>
                  );
                } else {
                  return (
                    <div key={cfg.field} className="flex flex-col gap-1">
                      <label className="text-xs font-semibold text-text-secondary">{cfg.label}</label>
                      <div className="relative flex items-center">
                        {cfg.prefix ? (
                          <span className="absolute left-3 text-sm text-text-secondary font-medium select-none pointer-events-none">
                            {cfg.prefix}
                          </span>
                        ) : null}
                        <input
                          type="number"
                          step="any"
                          placeholder="0"
                          className={`rounded-button border border-border py-2 text-sm text-text-primary w-full ${
                            cfg.prefix ? "pl-9" : "pl-3"
                          } ${cfg.suffix ? "pr-14" : "pr-3"}`}
                          value={formData[cfg.field]}
                          onChange={(e) => setFormData({ ...formData, [cfg.field]: e.target.value })}
                        />
                        {cfg.suffix ? (
                          <span className="absolute right-3 text-sm text-text-secondary font-medium select-none pointer-events-none">
                            {cfg.suffix}
                          </span>
                        ) : null}
                      </div>
                    </div>
                  );
                }
              })}
            </div>
          </div>
          <div className="flex gap-2">
            <Button onClick={handleSubmitForm} disabled={formLoading}>{formLoading ? (isEditing ? "Menyimpan..." : "Membuat...") : (isEditing ? "Simpan" : "Buat")}</Button>
            <Button variant="outline" onClick={closeForm}>Batal</Button>
          </div>
        </div>
      </Drawer>



      <Drawer
        open={showHistory}
        onClose={closeHistory}
        title="Riwayat Perubahan"
      >
        <div className="space-y-4">
          {historyError ? <div className="rounded-card bg-accent-red/10 px-3 py-2 text-xs text-accent-red">{historyError}</div> : null}
          {historyLoading ? (
            <p className="text-sm text-text-secondary">Memuat riwayat...</p>
          ) : historyRecords.length === 0 ? (
            <p className="text-sm text-text-secondary">Belum ada riwayat untuk warga ini.</p>
          ) : (
            <div className="space-y-3">
              {historyRecords.map((record) => {
                const latest = record.data_baru || {};
                const previous = record.data_lama || {};
                const actor = record.actor_name || record.user_id || "Sistem";
                const summary = latest.nama_lengkap || previous.nama_lengkap || selectedWarga?.name || "Warga";

                return (
                  <div key={record.id} className="rounded-card border border-border bg-white p-3">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <p className="text-sm font-semibold text-text-primary">{record.aksi}</p>
                        <p className="text-xs text-text-secondary">{actor} • {formatDate(record.created_at)}</p>
                      </div>
                      <Badge variant={record.aksi === "create" ? "success" : "info"}>{record.tabel}</Badge>
                    </div>
                    <div className="mt-3 grid gap-2 text-xs text-text-secondary">
                      <p><span className="font-semibold">Nama:</span> {summary}</p>
                      <p><span className="font-semibold">NIK:</span> {latest.nik || previous.nik || "-"}</p>
                      <p><span className="font-semibold">RT/RW:</span> {(latest.rt && latest.rw) ? `${latest.rt}/${latest.rw}` : (previous.rt && previous.rw ? `${previous.rt}/${previous.rw}` : "-")}</p>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
          <div className="flex gap-2">
            <Button variant="outline" onClick={closeHistory}>Tutup</Button>
          </div>
        </div>
      </Drawer>

      {previewImage ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
          onClick={() => setPreviewImage(null)}
        >
          <div className="relative max-w-full max-h-full">
            <button
              className="absolute top-4 right-4 text-white bg-black/60 hover:bg-black/80 rounded-full w-10 h-10 flex items-center justify-center text-xl font-bold transition-colors"
              onClick={() => setPreviewImage(null)}
            >
              ✕
            </button>
            <img
              src={previewImage}
              alt="Preview Dokumen"
              className="max-w-full max-h-[85vh] rounded-card object-contain shadow-2xl animate-scale-up"
              onClick={(e) => e.stopPropagation()}
            />
          </div>
        </div>
      ) : null}
    </AppShell>
  );
};

export default DataWargaPage;
