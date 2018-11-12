/** @format */

const fetch = window.fetch

import {toast} from 'react-toastify'
import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars

import RecordRow from './RecordRow'

export default function Home() {
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
      releases[0].assets.map(a => ({
        name: a.name,
        id: a.id,
        url: a.browser_download_url
      }))
    )
  }, [])

  return (
    <>
      <div>
        {downloads.map(d => (
          <div key={d.id}>
            <code>
              sudo curl{' '}
              <a href={d.url} target="_blank">
                {d.url}
              </a>{' '}
              > /usr/local/bin/gravity && sudo chmod /usr/local/bin/gravity
            </code>
          </div>
        ))}
      </div>
      <section>
        <h1>Objects</h1>
        <table>
          <tbody>
            {entries.map(entry => (
              <RecordRow key={entry.owner + '/' + entry.name} {...entry} />
            ))}
          </tbody>
        </table>
      </section>
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
