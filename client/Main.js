/** @format */

const fetch = window.fetch
const uniq = require('array-uniq')

import {ToastContainer, toast} from 'react-toastify'
import React, {useState, useEffect, useRef} from 'react' // eslint-disable-line no-unused-vars

import Portal from './Portal.js'

const service = {
  name: process.env.SERVICE_NAME || 'Planet',
  url: process.env.SERVICE_URL || 'https://github.com/fiatjaf/gravity',
  provider: {
    name: process.env.SERVICE_PROVIDER_NAME || 'gravity',
    url:
      process.env.SERVICE_PROVIDER_URL || 'https://github.com/fiatjaf/gravity'
  }
}

export default function Main() {
  let [downloads, setDownloads] = useState([])
  let [entries, setEntries] = useState([])

  async function loadEntries() {
    let entries = await fetchEntries()
    if (entries) setEntries(entries)
  }

  useEffect(loadEntries, [])

  useEffect(async () => {
    let releases = await fetchReleases()
    setDownloads(
      releases.assets.map(a => ({name: a.name, id: a.id, url: a.url}))
    )
  }, [])

  return (
    <>
      <ToastContainer />

      <Portal to="header > h1" clear>
        {service.name}
      </Portal>
      <Portal to="header aside .name" clear>
        {service.name.toLowerCase()}
      </Portal>

      <Portal to="#downloads">
        {downloads.map(d => (
          <div key={d.id}>
            <a href={d.url} target="_blank">
              {d.name}
            </a>{' '}
            <code>
              sudo curl {d.url} > /usr/local/bin/gravity && sudo chmod
              /usr/local/bin/gravity
            </code>
          </div>
        ))}
      </Portal>

      <table>
        <tbody>
          {entries.map(entry => (
            <RecordRow key={entry.owner + '/' + entry.name} {...entry} />
          ))}
        </tbody>
      </table>

      <Portal to="footer" clear>
        <p>
          <a href={service.provider.url}>{service.provider.name}</a>,{' '}
          {new Date().getFullYear()}
        </p>
      </Portal>
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

async function fetchReleases() {
  try {
    let res = await fetch(
      'https://api.github.com/repos/fiatjaf/gravity/releases?per_page=1'
    )
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  } catch (err) {
    console.error(err)
    toast('failed to fetch releases: ' + err.message, {
      type: 'error'
    })
  }
}
