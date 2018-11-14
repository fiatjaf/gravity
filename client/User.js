/** @format */

import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars

import RecordRow from './RecordRow'
import {fetchEntries} from './helpers'

export default function Home(props) {
  let {owner} = props.match.params
  let [entries, setEntries] = useState([])

  async function loadEntries() {
    let entries = await fetchEntries(owner)
    if (entries) setEntries(entries)
  }

  useEffect(loadEntries, [])

  return (
    <>
      <section>
        <h1>{owner}</h1>
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