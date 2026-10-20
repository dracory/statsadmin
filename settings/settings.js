const { createApp, ref, computed, onMounted } = Vue;

createApp({
    setup() {
        const ips = ref([]);
        const newIP = ref('');
        const loading = ref(false);
        const loaded = ref(false);
        const error = ref('');
        const success = ref('');

        // Bot IPs state
        const botIPs = ref([]);
        const newBotIP = ref('');
        const botLoading = ref(false);
        const botLoaded = ref(false);
        const botPage = ref(1);
        const botPerPage = ref(25);

        const botTotalPages = computed(() => Math.ceil(botIPs.value.length / botPerPage.value) || 1);
        const botPageStart = computed(() => (botPage.value - 1) * botPerPage.value);
        const botPageEnd = computed(() => Math.min(botPageStart.value + botPerPage.value, botIPs.value.length));
        const botPageItems = computed(() => botIPs.value.slice(botPageStart.value, botPageEnd.value));

        // Identify-bots state
        const candidates = ref([]);
        const identifying = ref(false);
        const deletingBots = ref(false);
        const deletingThreats = ref(false);
        const deletingOlderThan = ref(false);
        const deleteOlderThanDate = ref('');
        const totalRecords = ref(null);
        const olderCount = ref(null);

        const loadIPs = async () => {
            loading.value = true;
            error.value = '';
            try {
                const response = await fetch(urlLoadIPs, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({}),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    ips.value = data.data.ips || [];
                } else {
                    error.value = data.message || 'Failed to load IPs';
                }
            } catch (e) {
                error.value = 'Failed to load IPs';
            } finally {
                loading.value = false;
                loaded.value = true;
            }
        };

        const addIP = async () => {
            if (!newIP.value.trim()) return;
            loading.value = true;
            error.value = '';
            success.value = '';
            try {
                const response = await fetch(urlAddIP, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ip: newIP.value.trim() }),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    newIP.value = '';
                    success.value = 'IP added to exclusion list';
                    await loadIPs();
                } else {
                    error.value = data.message || 'Failed to add IP';
                }
            } catch (e) {
                error.value = 'Failed to add IP';
            } finally {
                loading.value = false;
            }
        };

        const removeIP = (ip) => {
            Notiflix.Confirm.show(
                'Stop Excluding',
                'Remove "' + ip + '" from the exclusion list? Future visits from this IP will be tracked again.',
                'Yes, remove it',
                'No, cancel',
                async () => {
                    loading.value = true;
                    error.value = '';
                    success.value = '';
                    try {
                        const response = await fetch(urlRemoveIP, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ ip: ip }),
                        });
                        const data = await response.json();
                        if (data.status === 'success') {
                            success.value = 'IP removed from exclusion list';
                            await loadIPs();
                        } else {
                            error.value = data.message || 'Failed to remove IP';
                        }
                    } catch (e) {
                        error.value = 'Failed to remove IP';
                    } finally {
                        loading.value = false;
                    }
                },
                () => {}
            );
        };

        const deleteVisitors = (ip) => {
            Notiflix.Confirm.show(
                'Delete Visitor Stats',
                'Permanently delete ALL visitor records from IP ' + ip + '? This cannot be undone.',
                'Yes, delete them',
                'No, cancel',
                async () => {
                    loading.value = true;
                    error.value = '';
                    success.value = '';
                    try {
                        const response = await fetch(urlDeleteVisitors, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ ip: ip }),
                        });
                        const data = await response.json();
                        if (data.status === 'success') {
                            success.value = data.message || 'Visitor records deleted';
                            await loadIPs();
                            await loadBots();
                        } else {
                            error.value = data.message || 'Failed to delete visitors';
                        }
                    } catch (e) {
                        error.value = 'Failed to delete visitors';
                    } finally {
                        loading.value = false;
                    }
                },
                () => {}
            );
        };

        // == Bot IPs ==

        const loadBots = async () => {
            botLoading.value = true;
            error.value = '';
            try {
                const response = await fetch(urlLoadBots, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({}),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    botIPs.value = data.data.bots || [];
                    botPage.value = 1;
                } else {
                    error.value = data.message || 'Failed to load bot IPs';
                }
            } catch (e) {
                error.value = 'Failed to load bot IPs';
            } finally {
                botLoading.value = false;
                botLoaded.value = true;
            }
        };

        const addBot = async () => {
            if (!newBotIP.value.trim()) return;
            botLoading.value = true;
            error.value = '';
            success.value = '';
            try {
                const response = await fetch(urlAddBot, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ip: newBotIP.value.trim() }),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    newBotIP.value = '';
                    success.value = 'IP added to bot list';
                    await loadBots();
                } else {
                    error.value = data.message || 'Failed to add bot IP';
                }
            } catch (e) {
                error.value = 'Failed to add bot IP';
            } finally {
                botLoading.value = false;
            }
        };

        const removeBot = (ip) => {
            Notiflix.Confirm.show(
                'Remove Bot IP',
                'Remove "' + ip + '" from the bot list?',
                'Yes, remove it',
                'No, cancel',
                async () => {
                    botLoading.value = true;
                    error.value = '';
                    success.value = '';
                    try {
                        const response = await fetch(urlRemoveBot, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ ip: ip }),
                        });
                        const data = await response.json();
                        if (data.status === 'success') {
                            success.value = 'IP removed from bot list';
                            await loadBots();
                        } else {
                            error.value = data.message || 'Failed to remove bot IP';
                        }
                    } catch (e) {
                        error.value = 'Failed to remove bot IP';
                    } finally {
                        botLoading.value = false;
                    }
                },
                () => {}
            );
        };

        const botPrevPage = () => {
            if (botPage.value > 1) botPage.value--;
        };
        const botNextPage = () => {
            if (botPage.value < botTotalPages.value) botPage.value++;
        };

        const deleteBotEntries = () => {
            Notiflix.Confirm.show(
                'Delete Bot Entries',
                'Permanently delete ALL visitor records tagged as bots (bot=yes)? This cannot be undone.',
                'Yes, delete them',
                'No, cancel',
                async () => {
                    deletingBots.value = true;
                    error.value = '';
                    success.value = '';
                    try {
                        const response = await fetch(urlDeleteBots, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({}),
                        });
                        const data = await response.json();
                        if (data.status === 'success') {
                            success.value = data.message || 'Bot entries deleted';
                            await loadBots();
                        } else {
                            error.value = data.message || 'Failed to delete bot entries';
                        }
                    } catch (e) {
                        error.value = 'Failed to delete bot entries';
                    } finally {
                        deletingBots.value = false;
                    }
                },
                () => {}
            );
        };

        const deleteThreatEntries = () => {
            Notiflix.Confirm.show(
                'Delete Threat Entries',
                'Permanently delete ALL visitor records tagged as threats (threat=yes)? This cannot be undone.',
                'Yes, delete them',
                'No, cancel',
                async () => {
                    deletingThreats.value = true;
                    error.value = '';
                    success.value = '';
                    try {
                        const response = await fetch(urlDeleteThreats, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({}),
                        });
                        const data = await response.json();
                        if (data.status === 'success') {
                            success.value = data.message || 'Threat entries deleted';
                            await loadBots();
                        } else {
                            error.value = data.message || 'Failed to delete threat entries';
                        }
                    } catch (e) {
                        error.value = 'Failed to delete threat entries';
                    } finally {
                        deletingThreats.value = false;
                    }
                },
                () => {}
            );
        };

        const loadStats = async (olderThan) => {
            try {
                const body = olderThan ? { older_than: olderThan } : {};
                const response = await fetch(urlLoadStats, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(body),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    totalRecords.value = data.data.total_records || 0;
                    if (olderThan) {
                        olderCount.value = data.data.older_count || 0;
                    } else {
                        olderCount.value = null;
                    }
                }
            } catch (e) {
                // Silently fail — count is informational only
            }
        };

        const onDateChange = () => {
            olderCount.value = null;
            if (deleteOlderThanDate.value) {
                loadStats(deleteOlderThanDate.value);
            }
        };

        const confirmDeleteOlderThan = () => {
            if (!deleteOlderThanDate.value) return;
            
            Notiflix.Confirm.show(
                'Delete Old Records',
                'Permanently delete ALL visitor records older than ' + deleteOlderThanDate.value + '? This cannot be undone.',
                'Yes, delete them',
                'No, cancel',
                async () => {
                    deletingOlderThan.value = true;
                    error.value = '';
                    success.value = '';
                    try {
                        const response = await fetch(urlDeleteVisitors, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ older_than: deleteOlderThanDate.value }),
                        });
                        const data = await response.json();
                        if (data.status === 'success') {
                            success.value = data.message || 'Old records deleted';
                            deleteOlderThanDate.value = '';
                            olderCount.value = null;
                            await loadStats();
                        } else {
                            error.value = data.message || 'Failed to delete old records';
                        }
                    } catch (e) {
                        error.value = 'Failed to delete old records';
                    } finally {
                        deletingOlderThan.value = false;
                    }
                },
                () => {}
            );
        };

        // == Identify Bots ==

        const identifyBots = async () => {
            identifying.value = true;
            error.value = '';
            success.value = '';
            try {
                const response = await fetch(urlIdentifyBots, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({}),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    candidates.value = data.data.candidates || [];
                    const autoAdded = data.data.auto_added_count || 0;
                    const unsure = candidates.value.length;
                    const total = autoAdded + unsure;
                    if (autoAdded > 0) {
                        await loadBots();
                    }
                    if (total === 0) {
                        Notiflix.Notify.info('No bot candidates found in visitor logs');
                    } else if (autoAdded > 0 && unsure > 0) {
                        Notiflix.Notify.success('Found ' + total + ': auto-added ' + autoAdded + ', ' + unsure + ' need review');
                    } else if (autoAdded > 0) {
                        Notiflix.Notify.success('Found ' + total + ': all ' + autoAdded + ' auto-added (high confidence)');
                    } else {
                        Notiflix.Notify.success('Found ' + total + ': all ' + unsure + ' need review');
                    }
                } else {
                    error.value = data.message || 'Failed to identify bots';
                }
            } catch (e) {
                error.value = 'Failed to identify bots';
            } finally {
                identifying.value = false;
            }
        };

        const addCandidate = async (ip) => {
            botLoading.value = true;
            error.value = '';
            const c = candidates.value.find(c => c.candidate_ip === ip);
            const reasons = c ? (c.reasons || []) : [];
            try {
                const response = await fetch(urlAddBot, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ip: ip, reasons: reasons }),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    // Remove the candidate from the list.
                    candidates.value = candidates.value.filter(c => c.candidate_ip !== ip);
                    await loadBots();
                } else {
                    error.value = data.message || 'Failed to add bot IP';
                }
            } catch (e) {
                error.value = 'Failed to add bot IP';
            } finally {
                botLoading.value = false;
            }
        };

        const addAllCandidates = async () => {
            const toAdd = candidates.value.filter(c => !c.already_listed);
            if (toAdd.length === 0) {
                Notiflix.Notify.info('No new candidates to add');
                return;
            }

            botLoading.value = true;
            error.value = '';
            let added = 0;
            let failed = 0;
            const addedIPs = [];
            for (const c of toAdd) {
                try {
                    const response = await fetch(urlAddBot, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ ip: c.candidate_ip, reasons: c.reasons || [] }),
                    });
                    const data = await response.json();
                    if (data.status === 'success') {
                        addedIPs.push(c.candidate_ip);
                        added++;
                    } else {
                        failed++;
                    }
                } catch (e) {
                    failed++;
                }
            }
            // Remove successfully added candidates from the list.
            if (addedIPs.length > 0) {
                candidates.value = candidates.value.filter(c => !addedIPs.includes(c.candidate_ip));
            }
            botLoading.value = false;
            await loadBots();

            if (added > 0 && failed === 0) {
                Notiflix.Notify.success('Added ' + added + ' bot IP(s)');
            } else if (added > 0 && failed > 0) {
                Notiflix.Notify.warning('Added ' + added + ', failed ' + failed);
            } else {
                Notiflix.Notify.failure('Failed to add bot IPs');
            }
        };

        const ipDetailsUrl = (ip) => {
            return urlIPDetailsBase + '&ip=' + encodeURIComponent(ip);
        };

        onMounted(() => {
            loadIPs();
            loadBots();
            loadStats();
        });

        return {
            ips,
            newIP,
            loading,
            loaded,
            error,
            success,
            loadIPs,
            addIP,
            removeIP,
            deleteVisitors,
            botIPs,
            newBotIP,
            botLoading,
            botLoaded,
            botPage,
            botPerPage,
            botTotalPages,
            botPageStart,
            botPageEnd,
            botPageItems,
            loadBots,
            addBot,
            removeBot,
            botPrevPage,
            botNextPage,
            candidates,
            identifying,
            identifyBots,
            addCandidate,
            addAllCandidates,
            deleteBotEntries,
            deleteThreatEntries,
            deletingBots,
            deletingThreats,
            deletingOlderThan,
            deleteOlderThanDate,
            confirmDeleteOlderThan,
            totalRecords,
            olderCount,
            onDateChange,
            ipDetailsUrl,
        };
    }
}).mount('#stats-settings-app');
