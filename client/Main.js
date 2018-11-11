/** @format */

const fetch = window.fetch
const uniq = require('array-uniq')

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
        <tbody>
          {entries.map(entry => (
            <RecordRow key={entry.owner + '/' + entry.name} {...entry} />
          ))}
        </tbody>
      </table>
    </>
  )
}

function RecordRow({owner, name, cid, note}) {
  let [nprovs, setNProvs] = useState(null)

  useEffect(() => {
    if (window.ipfs) {
      window.ipfs.dht
        .findprovs(cid)
        .catch(err => console.warn('error finding provs for ' + cid, err))
        .then(peerInfos => {
          setNProvs(uniq(peerInfos.map(p => p.ID).filter(x => x)).length)
        })
    }
  }, [])

  return (
    <tr>
      <td>
        {owner}/{name}
      </td>
      <td>
        <a target="_blank" href={`https://ipfs.io/ipfs/${cid}`}>
          {cid}
        </a>
      </td>
      {nprovs !== null && <td>{nprovs} providers</td>}
      <td>{note}</td>
    </tr>
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
