/** @format */

const hashbow = require('hashbow')

import {ToastContainer} from 'react-toastify'
import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars
import {BrowserRouter as Router, Route, Link} from 'react-router-dom'

import Portal from './Portal'
import Home from './Home'
import Record from './Record'
import User from './User'

export const service = {
  name: process.env.SERVICE_NAME || 'Planet',
  url: process.env.SERVICE_URL || 'https://github.com/fiatjaf/gravity',
  icon: process.env.ICON,
  provider: {
    name: process.env.SERVICE_PROVIDER_NAME || 'gravity',
    url:
      process.env.SERVICE_PROVIDER_URL || 'https://github.com/fiatjaf/gravity'
  }
}

export const GlobalContext = React.createContext({})

export default function Main() {
  let [nodeId, setNodeId] = useState(null)

  useEffect(() => {
    if (window.ipfs) {
      window.ipfs.id().then(info => {
        setNodeId(info.ID)
      })
    }
  }, [])

  return (
    <>
      <Router>
        <GlobalContext.Provider value={{nodeId}}>
          <ToastContainer />

          <Portal to="title" clear>
            {service.name} - IPFS Gravitational Body
          </Portal>
          <Portal to="header > h1" clear>
            <Link
              to="/"
              className="icon"
              dangerouslySetInnerHTML={{__html: service.icon}}
            />
            <Link to="/">{service.name}</Link>
          </Portal>
          <Portal to="header aside .name" clear>
            {service.name.toLowerCase()}
          </Portal>
          <Portal to=".bin-link" clear>
            <Link to="/fiatjaf/gravity">fiatjaf/gravity</Link>
          </Portal>

          <Route exact path="/" component={Home} />
          <Route path="/:x" component={Cleanup} />
          <Route path="/" component={Colorize} />
          <Route exact path="/:owner" component={User} />
          <Route path="/:owner/:name" component={Record} />

          <Portal to="footer .provider" clear>
            <>
              <a href={service.provider.url}>{service.provider.name}</a>,{' '}
              {new Date().getFullYear()}
            </>
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

function Colorize() {
  useEffect(() => {
    document.documentElement.style.setProperty(
      '--bg-color-1',
      hashbow(location.href.slice(0, 9))
    )
    document.documentElement.style.setProperty(
      '--bg-color-2',
      hashbow(location.href.slice(-9), 20, 30)
    )
  })

  return null
}
