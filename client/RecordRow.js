/** @format */

import React, {useState, useEffect, useContext} from 'react' // eslint-disable-line no-unused-vars
import {Link} from 'react-router-dom'

import {GlobalContext} from './Main'

export default function RecordRow({owner, name, cid, note, nstars}) {
  let {nodeId} = useContext(GlobalContext)

  let [nprovs, setNProvs] = useState(null)
  let [ishere, setIsHere] = useState(false)

  useEffect(() => {
    if (window.ipfs) {
      window.ipfs.dht
        .findprovs(cid)
        .then(peerInfos => {
          setNProvs(
            [...new Set(peerInfos.map(p => p.ID).filter(x => x))].length
          )

          for (let i = 0; i < peerInfos.length; i++) {
            if (peerInfos[i].ID === nodeId) {
              setIsHere(true)
            }
          }
        })
        .catch(err => console.warn('error finding provs for ' + cid, err))
    }
  }, [])

  return (
    <div className="record-row">
      <div className="address">
        <Link className="dirlink" to={`/${owner}`}>
          {owner}
        </Link>
        /
        <Link className="recordlink" to={`/${owner}/${name}`}>
          {name}
        </Link>
        {nstars !== 0 ? `★ ${nstars}` : ''}
      </div>
      <div className="cid">
        <a
          className="cidlink"
          target="_blank"
          href={`https://ipfs.io/ipfs/${cid}`}
        >
          {cid}
        </a>
      </div>
      <div className="note">{note}</div>
      <div>
        {nprovs !== null && (
          <span title={`${nprovs} providers for this object found.`}>
            {nprovs}
          </span>
        )}
      </div>
      <div>
        {nprovs !== null && (
          <span
            title={
              ishere
                ? 'This object is present in your local node.'
                : 'Object not found in your local node.'
            }
          >
            {ishere ? 'here' : '-'}
          </span>
        )}
      </div>
    </div>
  )
}
