/** @format */

import {ToastContainer} from 'react-toastify'
import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars
import {BrowserRouter as Router, Route, Link} from 'react-router-dom'

import Portal from './Portal'
import Home from './Home'
import Record from './Record'
import User from './User'

const service = {
  name: process.env.SERVICE_NAME || 'Planet',
  url: process.env.SERVICE_URL || 'https://github.com/fiatjaf/gravity',
  provider: {
    name: process.env.SERVICE_PROVIDER_NAME || 'gravity',
    url:
      process.env.SERVICE_PROVIDER_URL || 'https://github.com/fiatjaf/gravity'
  }
}

export const GlobalContext = React.createContext({})

export default function Main() {
  let [nodeId, setNodeId] = useState(null)

  useEffect(async () => {
    if (window.ipfs) {
      let info = await window.ipfs.id()
      setNodeId(info.ID)
    }
  })

  return (
    <>
      <Router>
        <GlobalContext.Provider value={{nodeId}}>
          <ToastContainer />

          <Portal to="title" clear>
            {service.name} - IPFS Gravitational Body
          </Portal>
          <Portal to="header > h1" clear>
            <Link to="/">{service.name}</Link>
          </Portal>
          <Portal to="header aside .name" clear>
            {service.name.toLowerCase()}
          </Portal>
          <Portal to=".bin-link" clear>
            <Link to="/fiatjaf/gravity-binaries">fiatjaf/gravity-binaries</Link>
          </Portal>

          <Route exact path="/" component={Home} />
          <Route path="/" component={Cleanup} />
          <Route exact path="/:owner" component={User} />
          <Route path="/:owner/:name" component={Record} />

          <Portal to="footer" clear>
            <p>
              <a href={service.provider.url}>{service.provider.name}</a>,{' '}
              {new Date().getFullYear()}
            </p>
          </Portal>
        </GlobalContext.Provider>
      </Router>
    </>
  )
}

function Cleanup() {
  return (
    <>
      <Portal to="body > header aside" clear />
      <Portal to="#how-to" clear />
    </>
  )
}
