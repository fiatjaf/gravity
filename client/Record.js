/** @format */

const fetch = window.fetch

import {toast} from 'react-toastify'
import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars

export default function Home(props) {
  let {owner, name} = props.match.params
  let [entry, setEntry] = useState(null)

  useEffect(
    async () => {
      let entry = await fetchEntry(owner, name)
      setEntry(entry)
    },
    [owner, name]
  )

  return (
    <>
      <main id="record">
        <header>
          <h1>
            {owner}/{name}
          </h1>
          <aside>{entry && <p>{entry.note}</p>}</aside>
        </header>
        {entry && (
          <iframe src={`https://cloudflare-ipfs.com/ipfs/${entry.cid}`} />
        )}
      </main>
    </>
  )
}

async function fetchEntry(owner, name) {
  try {
    let res = await fetch(`/${owner}/${name}`)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  } catch (err) {
    console.error(err)
    toast('failed to fetch entry: ' + err.message, {
      type: 'error'
    })
  }
}
