/** @format */

import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars

import RecordRow from './RecordRow'
import {fetchEntries} from './helpers'

export default function Home() {
  let [entries, setEntries] = useState([])

  async function loadEntries() {
    let entries = await fetchEntries()
    if (entries) setEntries(entries)
  }

  useEffect(loadEntries, [])

  return (
    <>
      <section>
        <h1>Objects</h1>
        <div id="records">
          {entries.map(entry => (
            <RecordRow key={entry.owner + '/' + entry.name} {...entry} />
          ))}
        </div>
      </section>
    </>
  )
}
