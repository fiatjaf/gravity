/** @format */

const uniq = require('array-uniq')

import React, {useState, useEffect, useContext} from 'react' // eslint-disable-line no-unused-vars
import {Link} from 'react-router-dom'

import {GlobalContext} from './Main'

export default function RecordRow({owner, name, cid, note}) {
  let {nodeId} = useContext(GlobalContext)

  let [nprovs, setNProvs] = useState(null)
  let [ishere, setIsHere] = useState(false)

  useEffect(() => {
    if (window.ipfs) {
      window.ipfs.dht
        .findprovs(cid)
        .catch(err => console.warn('error finding provs for ' + cid, err))
        .then(peerInfos => {
          setNProvs(uniq(peerInfos.map(p => p.ID).filter(x => x)).length)

          for (let i = 0; i < peerInfos.length; i++) {
            if (peerInfos[i].ID === nodeId) {
              setIsHere(true)
            }
          }
        })
    }
  }, [])

  return (
    <tr>
      <td>
        <Link to={`/${owner}`}>{owner}</Link>/
        <Link to={`/${owner}/${name}`}>{name}</Link>
      </td>
      <td>
        <a target="_blank" href={`https://ipfs.io/ipfs/${cid}`}>
          {cid}
        </a>
      </td>
      {nprovs !== null && <td>{nprovs} providers</td>}
      {nprovs !== null && <td>{ishere ? 'pinned here' : ''}</td>}
      <td>{note}</td>
    </tr>
  )
}
