const fetch = window.fetch

import {toast} from 'react-toastify'

export async function fetchEntries(owner = '') {
  try {
    let res = await fetch('/' + owner)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  } catch (err) {
    console.error(err)
    toast('failed to fetch entries: ' + err.message, {
      type: 'error'
    })
  }
}
