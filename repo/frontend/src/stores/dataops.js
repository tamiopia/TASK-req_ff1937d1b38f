import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

// Data Operations dashboard store.
//
// Note: the ingestion endpoints (/api/v1/internal/data/*) require HMAC-signed
// requests from internal clients. The dashboard exposes admin views via the
// regular session-cookie API surface only for legal-hold management; live
// ingestion control belongs to ops tooling, not the browser.
export const useDataOpsStore = defineStore('dataops', () => {
  const holds = ref([])

  async function fetchHolds() {
    const { data } = await axios.get('/api/v1/admin/legal-holds')
    holds.value = data.holds
    return data.holds
  }

  async function placeHold({ source_id, job_id, reason }) {
    const payload = { reason }
    if (source_id) payload.source_id = source_id
    if (job_id) payload.job_id = job_id
    const { data } = await axios.post('/api/v1/admin/legal-holds', payload)
    await fetchHolds()
    return data.hold
  }

  async function releaseHold(id) {
    await axios.delete(`/api/v1/admin/legal-holds/${id}`)
    holds.value = holds.value.filter((h) => h.id !== id)
  }

  return { holds, fetchHolds, placeHold, releaseHold }
})
