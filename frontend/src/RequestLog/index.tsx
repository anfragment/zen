import { CardList, Card, Tag, Collapse } from '@blueprintjs/core';
import { useEffect, useState } from 'react';

// eslint-disable-next-line import/no-relative-packages
import { EventsOn } from '../../wailsjs/runtime';

import './index.css';

interface Log {
  id: string;
  method: string;
  url: string;
  referer: string;
  filterName: string;
  rule: string;
  createdAt: Date;
}

export function RequestLog() {
  const [logs, setLogs] = useState<Log[]>([]);

  useEffect(
    () =>
      EventsOn('proxy:filter', (...data) => {
        setLogs((prevLogs) =>
          [
            {
              id: id(),
              method: data[0],
              url: data[1],
              referer: data[2],
              filterName: data[3],
              rule: data[4],
              createdAt: new Date(),
            },
            ...prevLogs,
          ].slice(0, 200),
        );
      }),
    [],
  );

  return (
    <div className="request-log">
      {logs.length === 0 ? (
        <p className="request-log__empty">
          Start browsing to see blocked requests.
        </p>
      ) : (
        <CardList compact>
          {logs.map((log) => (
            <RequestLogCard log={log} key={log.id} />
          ))}
        </CardList>
      )}
    </div>
  );
}

function RequestLogCard({ log }: { log: Log }) {
  const [isOpen, setIsOpen] = useState(false);

  const { hostname } = new URL(log.url, 'http://foo'); // setting base url helps with //hostname:port urls

  return (
    <>
      <Card
        key={log.id}
        className="request-log__card"
        interactive
        onClick={() => setIsOpen(!isOpen)}
      >
        <Tag minimal intent="danger">
          {hostname}
        </Tag>
        <div className="bp5-text-muted">
          {log.createdAt.toLocaleTimeString([], { timeStyle: 'short' })}
        </div>
      </Card>

      <Collapse isOpen={isOpen}>
        <Card className="request-log__card__details">
          <p className="request-log__card__details__value">
            <strong>Method: </strong>
            <Tag minimal intent="primary">
              {log.method}
            </Tag>
          </p>
          <p className="request-log__card__details__value">
            <strong>URL: </strong>
            {log.url}
          </p>
          {log.referer && (
            <p className="request-log__card__details__value">
              <strong>Referer: </strong>
              {log.referer}
            </p>
          )}
          <p className="request-log__card__details__value">
            <strong>Filter name: </strong>
            {log.filterName}
          </p>
          <p className="request-log__card__details__value">
            <strong>Rule: </strong>
            {log.rule}
          </p>
        </Card>
      </Collapse>
    </>
  );
}

function id(): string {
  return Math.random().toString(36).substr(2, 9);
}
