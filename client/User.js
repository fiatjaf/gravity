/** @format */

import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars

import RecordRow from './RecordRow'
import {fetchEntries} from './helpers'

export default function Home(props) {
  let {owner} = props.match.params
  let [entries, setEntries] = useState([])

  useEffect(() => {
    fetchEntries(owner).then(entries => {
      if (entries) setEntries(entries)
    })
  }, [])

  return (
    <>
      <section>
        <h1>{owner}</h1>
        <div id="records">
          {entries.map(entry => (
            <RecordRow key={entry.owner + '/' + entry.name} {...entry} />
          ))}
        </div>
      </section>
    </>
  )
}
