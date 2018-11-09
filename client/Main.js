/** @format */

const fetch = window.fetch

import {ToastContainer, toast} from 'react-toastify'
import React, {useState, useEffect, useRef} from 'react' // eslint-disable-line no-unused-vars

export default function Main() {
  let [entries, setEntries] = useState([])

  async function loadEntries() {
    let entries = await fetchEntries()
    if (entries) setEntries(entries)
  }

  useEffect(loadEntries, [])

  return (
    <>
      <ToastContainer />

      <table>
        {entries.map(({owner, name, cid}) => (
          <tr key={owner + '/' + name}>
            <td>
              {owner}/{name}
            </td>
            <td>{cid}</td>
          </tr>
        ))}
      </table>
    </>
  )
}

async function fetchEntries() {
  try {
    let res = await fetch('/')
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  } catch (err) {
    console.error(err)
    toast('failed to fetch entries: ' + err.message, {
      type: 'error'
    })
  }
}
