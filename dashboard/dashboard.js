const { createApp, ref, computed, onMounted } = Vue;

createApp({
    setup() {
        const conditions = ref([]);

        if (localStorage.getItem('stats_include_bots') === 'false') {
            if (!conditions.value.some(c => c.field === 'is_bot' && c.value === 'no')) {
                conditions.value.push({ field: 'is_bot', operator: 'equals', value: 'no' });
            }
        }

        const includeBots = computed(() => {
            return !conditions.value.some(c => c.field === 'is_bot' && c.value === 'no');
        });

        const totalVisitors = ref(0);
        const uniqueVisitors = ref(0);
        const periodLabel = ref('');
        const selectedPeriod = ref('last-7-days');
        const topPaths = ref([]);
        const topCountries = ref([]);
        const topBrowsers = ref([]);
        const topOS = ref([]);
        const topDeviceTypes = ref([]);
        const recentVisitors = ref([]);

        const toggleBots = () => {
            if (includeBots.value) {
                if (!conditions.value.some(c => c.field === 'is_bot' && c.value === 'no')) {
                    conditions.value.push({ field: 'is_bot', operator: 'equals', value: 'no' });
                }
                localStorage.setItem('stats_include_bots', 'false');
            } else {
                conditions.value = conditions.value.filter(c => c.field !== 'is_bot');
                localStorage.setItem('stats_include_bots', 'true');
            }
            loadDashboard();
        };

        const periodOptions = [
            { value: 'today', label: 'Today' },
            { value: 'yesterday', label: 'Yesterday' },
            { value: 'last-7-days', label: 'Last 7 days' },
            { value: 'this-month', label: 'This month' },
            { value: 'last-month', label: 'Last month' },
            { value: 'all-time', label: 'All time' },
        ];

        const avgPerDay = computed(() => {
            const opt = periodOptions.find(o => o.value === selectedPeriod.value);
            if (!opt) return '0';
            let days = 1;
            if (selectedPeriod.value === 'last-7-days') days = 7;
            else if (selectedPeriod.value === 'this-month' || selectedPeriod.value === 'last-month') days = 30;
            else if (selectedPeriod.value === 'all-time') return '—';
            if (totalVisitors.value === 0) return '0';
            return (totalVisitors.value / days).toFixed(1);
        });

        const loadDashboard = async () => {
            try {
                const response = await fetch(urlLoadDashboard, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ period: selectedPeriod.value, conditions: conditions.value }),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    const d = data.data;
                    totalVisitors.value = d.total_visitors || 0;
                    uniqueVisitors.value = d.unique_visitors || 0;
                    periodLabel.value = d.period_label || '';
                    topPaths.value = d.top_paths || [];
                    topCountries.value = d.top_countries || [];
                    topBrowsers.value = d.top_browsers || [];
                    topOS.value = d.top_os || [];
                    topDeviceTypes.value = d.top_device_types || [];
                    recentVisitors.value = d.recent_visitors || [];
                } else {
                    Notiflix.Notify.failure(data.message || 'Failed to load dashboard');
                }
            } catch (error) {
                Notiflix.Notify.failure('Failed to load dashboard');
            }
        };

        const exportCSV = async () => {
            try {
                const response = await fetch(urlLoadDashboard.replace('action=load-dashboard', 'action=export-csv'), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ period: selectedPeriod.value, conditions: conditions.value }),
                });
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `visitors_${selectedPeriod.value}.csv`;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
            } catch (error) {
                Notiflix.Notify.failure('Failed to export CSV');
            }
        };

        const ipDetailsUrl = (ip) => {
            return urlIPDetailsBase + '&ip=' + encodeURIComponent(ip);
        };

        // Build a URL to the visitors page with a path-contains filter
        // pre-applied. The path label is stored as "[METHOD] /uri" so we
        // strip the method prefix to get just the path for filtering.
        const visitorsPathFilterUrl = (pathLabel) => {
            let pathValue = pathLabel;
            // Strip "[METHOD] " prefix if present.
            const match = pathLabel.match(/^\[[A-Z]+\]\s*(.*)$/);
            if (match) {
                pathValue = match[1];
            }
            return visitorsFilterUrl('path_contains', pathValue);
        };

        // Build a URL to the visitors page with a generic filter pre-applied.
        // Used for country, browser, os, device_type fields.
        const visitorsFilterUrl = (field, value) => {
            const filters = JSON.stringify([{ field: field, operator: 'equals', value: value }]);
            return urlVisitorsBase + '&filters=' + encodeURIComponent(filters);
        };

        onMounted(() => {
            loadDashboard();
        });

        return {
            includeBots,
            toggleBots,
            totalVisitors,
            uniqueVisitors,
            periodLabel,
            selectedPeriod,
            periodOptions,
            avgPerDay,
            topPaths,
            topCountries,
            topBrowsers,
            topOS,
            topDeviceTypes,
            recentVisitors,
            loadDashboard,
            exportCSV,
            ipDetailsUrl,
            visitorsPathFilterUrl,
            visitorsFilterUrl,
        };
    }
}).mount('#stats-dashboard-app');
