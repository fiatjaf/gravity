/** @format */

const fetch = window.fetch
const md = require('markdown-it')({
  html: true,
  linkify: true
})

import {toast} from 'react-toastify'
import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars
import {Link} from 'react-router-dom'

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
            <Link className="dirlink" to={`/${owner}`}>
              {owner}
            </Link>
            /{name}
            {entry && entry.nstars !== 0 ? ` ★ ${entry.nstars}` : ''}
          </h1>
          <aside>{entry && entry.note && <p>{entry.note}</p>}</aside>
        </header>
        {entry && (
          <>
            {entry.body && (
              <div
                className="body"
                dangerouslySetInnerHTML={{__html: md.render(entry.body)}}
              />
            )}
            <iframe src={`https://cloudflare-ipfs.com/ipfs/${entry.cid}`} />
            {entry.history && (
              <div id="history">
                <h3>Versions</h3>
                <table>
                  <tbody>
                    {entry.history.map(({cid, date}) => (
                      <tr
                        key={date}
                        className={cid === entry.cid ? 'current' : ''}
                      >
                        <td>
                          <a
                            className="cidlink"
                            href={`https://ipfs.io/ipfs/${cid}`}
                            target="_blank"
                          >
                            {cid}
                          </a>
                        </td>
                        <td>{date}</td>
                        <td>{cid === entry.cid ? 'CURRENT' : ''}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        )}
      </main>
    </>
  )
}

async function fetchEntry(owner, name) {
  try {
    let res = await fetch(`/${owner}/${name}?full=1`)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  } catch (err) {
    console.error(err)
    toast('failed to fetch entry: ' + err.message, {
      type: 'error'
    })
  }
}
